package unity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewPackageJSON(t *testing.T) {
	pkg := NewPackageJSON("com.example.test", "1.2.3", "Test package")

	if pkg.Name != "com.example.test" {
		t.Errorf("Name: got %q, want %q", pkg.Name, "com.example.test")
	}
	if pkg.Version != "1.2.3" {
		t.Errorf("Version: got %q, want %q", pkg.Version, "1.2.3")
	}
	if pkg.Description != "Test package" {
		t.Errorf("Description: got %q", pkg.Description)
	}
}

func TestNewPackageJSON_DefaultVersion(t *testing.T) {
	pkg := NewPackageJSON("com.example.test", "", "")
	if pkg.Version != "0.0.0" {
		t.Errorf("default version: got %q, want %q", pkg.Version, "0.0.0")
	}
}

func TestWritePackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := NewPackageJSON("com.example.test", "1.0.0", "A test")

	if err := WritePackageJSON(dir, pkg); err != nil {
		t.Fatalf("WritePackageJSON: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}

	var parsed PackageJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parsing package.json: %v", err)
	}

	if parsed.Name != "com.example.test" {
		t.Errorf("parsed name: got %q", parsed.Name)
	}
}

func TestReadPackageJSON(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"name":"com.example.test","version":"2.0.0","displayName":"Test","description":"desc","unity":"2021.3"}`)
	os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)

	pkg, err := ReadPackageJSON(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("ReadPackageJSON: %v", err)
	}
	if pkg.Name != "com.example.test" {
		t.Errorf("Name: got %q", pkg.Name)
	}
	if pkg.Version != "2.0.0" {
		t.Errorf("Version: got %q", pkg.Version)
	}
}
