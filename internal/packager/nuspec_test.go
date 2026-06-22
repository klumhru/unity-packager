package packager

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeFramework(t *testing.T) {
	cases := map[string]string{
		".NETStandard2.0":    "netstandard20",
		"netstandard2.0":     "netstandard20",
		".NETFramework4.6.1": "net461",
		"net461":             "net461",
		".NETCoreApp3.1":     "netcoreapp31",
		"netcoreapp3.1":      "netcoreapp31",
		"":                   "",
	}
	for in, want := range cases {
		if got := normalizeFramework(in); got != want {
			t.Errorf("normalizeFramework(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	cases := map[string]string{
		"1.2.3":        "1.2.3",
		"[1.2.3]":      "1.2.3",
		"[1.2.3, )":    "1.2.3",
		"[1.0.0, 2.0)": "1.0.0",
		"(, 2.0)":      "",
		"":             "",
		"1.0.*":        "",
	}
	for in, want := range cases {
		if got := resolveVersion(in); got != want {
			t.Errorf("resolveVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSkipNuGetDependency(t *testing.T) {
	skip := []string{
		"NETStandard.Library", "Microsoft.NETCore.Platforms", "Microsoft.NETFramework.ReferenceAssemblies",
		"System.Memory", "System.Buffers", "System.Runtime.CompilerServices.Unsafe",
		"System.Numerics.Vectors", "System.Threading.Tasks.Extensions", "System.ValueTuple",
	}
	keep := []string{
		"Grpc.Core.Api", "Newtonsoft.Json",
		"System.Collections.Immutable", "System.Reflection.Metadata",
	}
	for _, id := range skip {
		if !skipNuGetDependency(id) {
			t.Errorf("skipNuGetDependency(%q) = false, want true", id)
		}
	}
	for _, id := range keep {
		if skipNuGetDependency(id) {
			t.Errorf("skipNuGetDependency(%q) = true, want false", id)
		}
	}
}

func TestDependenciesForFramework(t *testing.T) {
	ns := &nuspec{}
	ns.Metadata.Dependencies.Groups = []nuspecGroup{
		{TargetFramework: ".NETStandard1.3", Deps: []nuspecDep{{ID: "Old", Version: "1.0.0"}}},
		{TargetFramework: ".NETStandard2.0", Deps: []nuspecDep{{ID: "New", Version: "2.0.0"}}},
	}

	// Exact match.
	got := ns.dependenciesForFramework("netstandard2.0")
	if len(got) != 1 || got[0].ID != "New" {
		t.Fatalf("exact match: got %+v", got)
	}

	// Nearest-compatible fallback: target 2.1 has no exact group, pick 2.0.
	got = ns.dependenciesForFramework("netstandard2.1")
	if len(got) != 1 || got[0].ID != "New" {
		t.Fatalf("nearest match: got %+v", got)
	}

	// Below-all fallback: target 1.0 has no <= group.
	if got = ns.dependenciesForFramework("netstandard1.0"); len(got) != 0 {
		t.Fatalf("below-all: got %+v", got)
	}
}

func TestDependenciesForFrameworkFlat(t *testing.T) {
	ns := &nuspec{}
	ns.Metadata.Dependencies.Deps = []nuspecDep{{ID: "Flat", Version: "1.0.0"}}
	got := ns.dependenciesForFramework("netstandard2.0")
	if len(got) != 1 || got[0].ID != "Flat" {
		t.Fatalf("flat deps: got %+v", got)
	}
}

// writeNupkg builds a minimal .nupkg zip with the given nuspec body and lib
// files (map of "framework/file" -> contents).
func writeNupkg(t *testing.T, dir, name, nuspecBody string, libFiles map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	w, err := zw.Create("package.nuspec")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(nuspecBody)); err != nil {
		t.Fatal(err)
	}

	for rel, content := range libFiles {
		w, err := zw.Create("lib/" + rel)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseNuspec(t *testing.T) {
	dir := t.TempDir()
	body := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>Demo</id>
    <dependencies>
      <group targetFramework=".NETStandard2.0">
        <dependency id="System.Memory" version="4.5.3" />
      </group>
    </dependencies>
  </metadata>
</package>`
	path := writeNupkg(t, dir, "demo.nupkg", body, nil)

	ns, err := parseNuspec(path)
	if err != nil {
		t.Fatal(err)
	}
	deps := ns.dependenciesForFramework("netstandard2.0")
	if len(deps) != 1 || deps[0].ID != "System.Memory" || deps[0].Version != "4.5.3" {
		t.Fatalf("got %+v", deps)
	}
}

func TestBestLibFramework(t *testing.T) {
	dir := t.TempDir()
	path := writeNupkg(t, dir, "lib.nupkg", "<package/>", map[string]string{
		"netstandard1.3/A.dll": "x",
		"netstandard2.0/A.dll": "y",
		"net461/A.dll":         "z",
	})
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if got := bestLibFramework(r, "netstandard2.0"); got != "netstandard2.0" {
		t.Errorf("exact: got %q", got)
	}
	if got := bestLibFramework(r, "netstandard2.1"); got != "netstandard2.0" {
		t.Errorf("nearest: got %q", got)
	}

	// No compatible netstandard.
	path2 := writeNupkg(t, dir, "lib2.nupkg", "<package/>", map[string]string{
		"netstandard2.1/A.dll": "x",
	})
	r2, err := zip.OpenReader(path2)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	if got := bestLibFramework(r2, "netstandard2.0"); got != "" {
		t.Errorf("incompatible: got %q, want empty", got)
	}
}
