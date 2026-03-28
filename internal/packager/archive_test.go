package packager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klumhru/unity-packager/internal/config"
)

func TestProcessArchive_ZipWithPackageJSON(t *testing.T) {
	// Create a zip archive with a package.json (unity package)
	zipPath := createTestZip(t, map[string]string{
		"package.json":       `{"name":"com.test.pkg","version":"1.0.0"}`,
		"Runtime/Foo.cs":     "namespace Test { class Foo {} }",
		"Runtime/Foo.cs.meta": "existing meta",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer srv.Close()

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "Packages"), 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})
	p.cache = &Cache{enabled: false}

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name: "com.test.pkg",
				Type: config.Archive,
				URL:  srv.URL + "/test.zip",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(projectDir, "Packages", "com.test.pkg")

	// package.json should be copied from archive (unity package mode)
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should exist")
	}

	// Runtime/Foo.cs should be copied
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Foo.cs")); os.IsNotExist(err) {
		t.Error("Runtime/Foo.cs should exist")
	}
}

func TestProcessArchive_ZipRawPackage(t *testing.T) {
	// Create a zip without package.json (raw source)
	zipPath := createTestZip(t, map[string]string{
		"Foo.cs": "namespace MyLib { class Foo {} }",
		"Bar.cs": "namespace MyLib { class Bar {} }",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer srv.Close()

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "Packages"), 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})
	p.cache = &Cache{enabled: false}

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name:        "com.test.raw",
				Type:        config.Archive,
				URL:         srv.URL + "/test.zip",
				Version:     "1.0.0",
				Description: "Test raw package",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(projectDir, "Packages", "com.test.raw")

	// package.json should be generated
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should be generated")
	}

	// Source should be under Runtime/
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Foo.cs")); os.IsNotExist(err) {
		t.Error("Runtime/Foo.cs should exist")
	}

	// Asmdef should be generated
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "com.test.raw.asmdef")); os.IsNotExist(err) {
		t.Error("asmdef should be generated")
	}
}

func TestProcessArchive_TarGz(t *testing.T) {
	tgzPath := createTestTarGz(t, map[string]string{
		"package.json":   `{"name":"com.test.tgz","version":"2.0.0"}`,
		"Runtime/Baz.cs": "namespace Baz { class Baz {} }",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tgzPath)
	}))
	defer srv.Close()

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "Packages"), 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})
	p.cache = &Cache{enabled: false}

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name: "com.test.tgz",
				Type: config.Archive,
				URL:  srv.URL + "/test.tar.gz",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(projectDir, "Packages", "com.test.tgz")
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should exist")
	}
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Baz.cs")); os.IsNotExist(err) {
		t.Error("Runtime/Baz.cs should exist")
	}
}

func TestProcessArchive_WrappedTopLevelDir(t *testing.T) {
	// Simulate archives that wrap everything in a top-level folder
	zipPath := createTestZip(t, map[string]string{
		"firebase-sdk-12.0.0/package.json":       `{"name":"com.test.firebase","version":"12.0.0"}`,
		"firebase-sdk-12.0.0/Runtime/Auth.cs":    "namespace Firebase.Auth { class Auth {} }",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer srv.Close()

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "Packages"), 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})
	p.cache = &Cache{enabled: false}

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name: "com.test.firebase",
				Type: config.Archive,
				URL:  srv.URL + "/firebase.zip",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(projectDir, "Packages", "com.test.firebase")
	// Should unwrap the top-level dir and find package.json
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should exist (auto-unwrapped from top-level dir)")
	}
}

func TestProcessArchive_WithPathAndExclude(t *testing.T) {
	zipPath := createTestZip(t, map[string]string{
		"src/lib/Foo.cs":      "namespace Lib { class Foo {} }",
		"src/lib/Tests/T.cs":  "namespace Lib.Tests { class T {} }",
		"src/other/Other.cs":  "namespace Other { class Other {} }",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer srv.Close()

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "Packages"), 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})
	p.cache = &Cache{enabled: false}

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name:    "com.test.subpath",
				Type:    config.Archive,
				URL:     srv.URL + "/test.zip",
				Path:    "src/lib",
				Exclude: []string{"Tests/**"},
				Version: "1.0.0",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(projectDir, "Packages", "com.test.subpath")

	// Foo.cs should be under Runtime (raw mode since no package.json)
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Foo.cs")); os.IsNotExist(err) {
		t.Error("Runtime/Foo.cs should exist")
	}

	// Tests should be excluded
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Tests")); !os.IsNotExist(err) {
		t.Error("Tests/ should be excluded")
	}
}

func TestUnwrapSingleDir(t *testing.T) {
	// Single subdirectory — should unwrap
	dir := t.TempDir()
	inner := filepath.Join(dir, "wrapper")
	os.MkdirAll(inner, 0755)
	os.WriteFile(filepath.Join(inner, "file.txt"), []byte("hi"), 0644)

	result := unwrapSingleDir(dir)
	if result != inner {
		t.Errorf("expected %s, got %s", inner, result)
	}

	// Multiple entries — should not unwrap
	dir2 := t.TempDir()
	os.MkdirAll(filepath.Join(dir2, "a"), 0755)
	os.WriteFile(filepath.Join(dir2, "b.txt"), []byte("hi"), 0644)

	result2 := unwrapSingleDir(dir2)
	if result2 != dir2 {
		t.Errorf("should not unwrap when multiple entries exist")
	}
}

func TestTryExtract_DetectsFormat(t *testing.T) {
	// Test zip detection by magic bytes
	zipPath := createTestZip(t, map[string]string{"test.txt": "hello"})
	destDir := t.TempDir()
	if err := tryExtract(zipPath, destDir); err != nil {
		t.Fatalf("tryExtract zip: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "test.txt")); os.IsNotExist(err) {
		t.Error("zip extraction via magic bytes failed")
	}

	// Test tar.gz detection by magic bytes
	tgzPath := createTestTarGz(t, map[string]string{"test2.txt": "world"})
	destDir2 := t.TempDir()
	if err := tryExtract(tgzPath, destDir2); err != nil {
		t.Fatalf("tryExtract tgz: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir2, "test2.txt")); os.IsNotExist(err) {
		t.Error("tgz extraction via magic bytes failed")
	}
}

// helpers

func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	createdDirs := make(map[string]bool)
	for name, content := range files {
		// Ensure parent directories exist as proper directory entries
		parts := strings.Split(name, "/")
		for i := 1; i < len(parts); i++ {
			dirName := strings.Join(parts[:i], "/") + "/"
			if !createdDirs[dirName] {
				hdr := &zip.FileHeader{Name: dirName}
				hdr.SetMode(0755 | os.ModeDir)
				w.CreateHeader(hdr)
				createdDirs[dirName] = true
			}
		}

		hdr := &zip.FileHeader{Name: name}
		hdr.SetMode(0644)
		fw, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	w.Close()
	return path
}

func createTestTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	for name, content := range files {
		// Write parent directories
		parts := strings.Split(name, "/")
		for i := 1; i < len(parts); i++ {
			dirName := strings.Join(parts[:i], "/") + "/"
			tw.WriteHeader(&tar.Header{
				Name:     dirName,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			})
		}

		tw.WriteHeader(&tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0644,
		})
		tw.Write([]byte(content))
	}

	tw.Close()
	gz.Close()
	return path
}
