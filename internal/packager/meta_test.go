package packager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateGUID_Deterministic(t *testing.T) {
	g1 := GenerateGUID("com.example.pkg", "Runtime/Foo.cs")
	g2 := GenerateGUID("com.example.pkg", "Runtime/Foo.cs")
	if g1 != g2 {
		t.Errorf("GUIDs not deterministic: %s != %s", g1, g2)
	}
	if len(g1) != 32 {
		t.Errorf("GUID length: got %d, want 32", len(g1))
	}
}

func TestGenerateGUID_DifferentInputs(t *testing.T) {
	g1 := GenerateGUID("com.example.a", "file.cs")
	g2 := GenerateGUID("com.example.b", "file.cs")
	if g1 == g2 {
		t.Error("different packages should produce different GUIDs")
	}

	g3 := GenerateGUID("com.example.a", "file1.cs")
	g4 := GenerateGUID("com.example.a", "file2.cs")
	if g3 == g4 {
		t.Error("different paths should produce different GUIDs")
	}
}

func TestWriteMetaFile_Directory(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "TestFolder.meta")

	if err := WriteMetaFile(metaPath, "abcdef1234567890abcdef1234567890", true); err != nil {
		t.Fatalf("WriteMetaFile: %v", err)
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("reading meta: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "folderAsset: yes") {
		t.Error("directory meta should contain folderAsset: yes")
	}
	if !strings.Contains(content, "guid: abcdef1234567890abcdef1234567890") {
		t.Error("meta should contain the GUID")
	}
}

func TestWriteMetaFile_DLL(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "Foo.dll.meta")

	if err := WriteMetaFile(metaPath, "abcdef1234567890abcdef1234567890", false); err != nil {
		t.Fatalf("WriteMetaFile: %v", err)
	}

	data, _ := os.ReadFile(metaPath)
	content := string(data)
	if !strings.Contains(content, "PluginImporter") {
		t.Error("DLL meta should use PluginImporter")
	}
}

func TestWriteMetaFile_CSharp(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "Foo.cs.meta")

	if err := WriteMetaFile(metaPath, "abcdef1234567890abcdef1234567890", false); err != nil {
		t.Fatalf("WriteMetaFile: %v", err)
	}

	data, _ := os.ReadFile(metaPath)
	content := string(data)
	if !strings.Contains(content, "MonoImporter") {
		t.Error("CS meta should use MonoImporter")
	}
}

func TestGenerateMetaFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some files and dirs
	os.MkdirAll(filepath.Join(dir, "Runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "Runtime", "Foo.cs"), []byte("class Foo {}"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	if err := GenerateMetaFiles(dir, "com.example.test"); err != nil {
		t.Fatalf("GenerateMetaFiles: %v", err)
	}

	// Check meta files exist
	for _, path := range []string{
		filepath.Join(dir, "Runtime.meta"),
		filepath.Join(dir, "Runtime", "Foo.cs.meta"),
		filepath.Join(dir, "package.json.meta"),
	} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected meta file %s to exist", path)
		}
	}
}

func TestGenerateMetaFiles_SkipsExisting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Foo.cs"), []byte("class Foo {}"), 0644)

	// Pre-create a meta file with custom content
	metaPath := filepath.Join(dir, "Foo.cs.meta")
	os.WriteFile(metaPath, []byte("custom content"), 0644)

	if err := GenerateMetaFiles(dir, "com.example.test"); err != nil {
		t.Fatalf("GenerateMetaFiles: %v", err)
	}

	// Should not overwrite
	data, _ := os.ReadFile(metaPath)
	if string(data) != "custom content" {
		t.Error("existing meta file was overwritten")
	}
}
