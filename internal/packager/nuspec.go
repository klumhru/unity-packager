package packager

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// nuspec is a minimal model of a .nuspec file, capturing only the dependency
// information needed for transitive resolution.
type nuspec struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		Dependencies struct {
			// Flat dependencies (no targetFramework grouping).
			Deps []nuspecDep `xml:"dependency"`
			// Framework-specific dependency groups.
			Groups []nuspecGroup `xml:"group"`
		} `xml:"dependencies"`
	} `xml:"metadata"`
}

type nuspecGroup struct {
	TargetFramework string      `xml:"targetFramework,attr"`
	Deps            []nuspecDep `xml:"dependency"`
}

type nuspecDep struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
}

// parseNuspec reads the root-level .nuspec entry from a .nupkg archive.
func parseNuspec(nupkgPath string) (*nuspec, error) {
	r, err := zip.OpenReader(nupkgPath)
	if err != nil {
		return nil, fmt.Errorf("opening nupkg: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// .nuspec lives at the archive root.
		if strings.Contains(f.Name, "/") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.Name), ".nuspec") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		var ns nuspec
		if err := xml.Unmarshal(data, &ns); err != nil {
			return nil, fmt.Errorf("parsing nuspec: %w", err)
		}
		return &ns, nil
	}

	return nil, fmt.Errorf("no .nuspec found in %s", nupkgPath)
}

// dependenciesForFramework returns the dependency list applicable to the given
// target framework. Flat (ungrouped) dependencies apply to every framework.
// For grouped dependencies it prefers an exact framework match, then a group
// with no targetFramework (applies to all), then the nearest-compatible
// netstandard group.
func (ns *nuspec) dependenciesForFramework(framework string) []nuspecDep {
	g := ns.Metadata.Dependencies.Groups
	if len(g) == 0 {
		return ns.Metadata.Dependencies.Deps
	}

	target := normalizeFramework(framework)

	// Exact match.
	for _, grp := range g {
		if normalizeFramework(grp.TargetFramework) == target {
			return grp.Deps
		}
	}

	// Group that applies to all frameworks.
	for _, grp := range g {
		if strings.TrimSpace(grp.TargetFramework) == "" {
			return grp.Deps
		}
	}

	// Nearest-compatible netstandard group (version <= target).
	if tv, ok := netstandardVersion(target); ok {
		bestVer := -1.0
		var best []nuspecDep
		for _, grp := range g {
			gv, ok := netstandardVersion(normalizeFramework(grp.TargetFramework))
			if ok && gv <= tv && gv > bestVer {
				bestVer = gv
				best = grp.Deps
			}
		}
		if best != nil {
			return best
		}
	}

	return nil
}

// normalizeFramework canonicalizes a target framework moniker so that long and
// short forms compare equal, e.g. ".NETStandard2.0" and "netstandard2.0" both
// become "netstandard20", and ".NETFramework4.6.1" and "net461" both become
// "net461".
func normalizeFramework(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, ".")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "-", "")
	if rest, ok := strings.CutPrefix(s, "netframework"); ok {
		s = "net" + rest
	}
	return s
}

// netstandardVersion extracts the numeric version from a normalized netstandard
// moniker, e.g. "netstandard20" -> 2.0. Returns false for non-netstandard input.
func netstandardVersion(normalized string) (float64, bool) {
	rest, ok := strings.CutPrefix(normalized, "netstandard")
	if !ok || rest == "" {
		return 0, false
	}
	// "20" -> "2.0", "21" -> "2.1", "13" -> "1.3"
	if len(rest) >= 2 {
		rest = rest[:1] + "." + rest[1:]
	}
	v, err := strconv.ParseFloat(rest, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// resolveVersion turns a NuGet dependency version range into a single concrete
// version, using the lowest applicable version (NuGet's default restore
// behavior). Returns "" when no concrete lower bound is available (open ranges
// or floating versions), in which case the dependency is skipped.
func resolveVersion(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}
	// Plain version (e.g. "1.2.3") means ">= 1.2.3".
	if !strings.ContainsAny(spec, "[]()") {
		if strings.ContainsAny(spec, "*") {
			return "" // floating version, cannot resolve offline
		}
		return spec
	}
	// Range form: [min, max], (min, max), [exact]. Take the lower bound token.
	inner := strings.Trim(spec, "[]()")
	lower := strings.TrimSpace(strings.SplitN(inner, ",", 2)[0])
	if lower == "" || strings.ContainsAny(lower, "*") {
		return ""
	}
	return lower
}

// unityProvidedPackages are exact NuGet ids whose assemblies Unity's runtime
// (the .NET Standard 2.0/2.1 profile) already ships. Bundling them again causes
// "duplicate assembly" / type-conflict errors in the editor, so they are skipped
// during transitive resolution. Matched by exact id (not prefix) — packages like
// System.Collections.Immutable and System.Reflection.Metadata are NOT provided
// by Unity and must still be resolved.
var unityProvidedPackages = map[string]bool{
	"system.memory":                          true,
	"system.buffers":                         true,
	"system.numerics.vectors":                true,
	"system.runtime.compilerservices.unsafe": true,
	"system.threading.tasks.extensions":      true,
	"system.valuetuple":                      true,
}

// skipNuGetDependency reports whether a dependency id refers to a framework or
// runtime meta-package that Unity's runtime already provides. Pulling these in
// would cause duplicate-assembly conflicts.
func skipNuGetDependency(id string) bool {
	lid := strings.ToLower(id)
	if lid == "netstandard.library" || unityProvidedPackages[lid] {
		return true
	}
	for _, prefix := range []string{
		"microsoft.netcore.",
		"microsoft.netframework.",
		"microsoft.aspnetcore.app.",
	} {
		if strings.HasPrefix(lid, prefix) {
			return true
		}
	}
	return false
}
