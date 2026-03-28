package packager

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestIsLegalFile(t *testing.T) {
	yes := []string{
		"LICENSE", "LICENSE.md", "LICENSE.txt",
		"license", "License.md",
		"LICENCE", "Licence.txt",
		"README", "README.md", "Readme.txt",
		"NOTICE", "NOTICE.md", "NOTICE.txt",
		"ThirdPartyNotices.txt",
	}
	for _, name := range yes {
		if !isLegalFile(name) {
			t.Errorf("isLegalFile(%q) = false, want true", name)
		}
	}

	no := []string{
		"main.go", "package.json", "Foo.cs", "LICENSE.bak",
		"LICENSING.md", "README.html",
	}
	for _, name := range no {
		if isLegalFile(name) {
			t.Errorf("isLegalFile(%q) = true, want false", name)
		}
	}
}

func TestCopyLegalFiles(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create source files
	os.WriteFile(filepath.Join(srcDir, "LICENSE"), []byte("MIT License"), 0644)
	os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# Hello"), 0644)
	os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)

	if err := CopyLegalFiles(srcDir, destDir); err != nil {
		t.Fatalf("CopyLegalFiles: %v", err)
	}

	// LICENSE and README.md should be copied
	data, err := os.ReadFile(filepath.Join(destDir, "LICENSE"))
	if err != nil {
		t.Error("LICENSE should be copied")
	} else if string(data) != "MIT License" {
		t.Errorf("LICENSE content: got %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(destDir, "README.md"))
	if err != nil {
		t.Error("README.md should be copied")
	} else if string(data) != "# Hello" {
		t.Errorf("README.md content: got %q", string(data))
	}

	// main.go should NOT be copied
	if _, err := os.Stat(filepath.Join(destDir, "main.go")); !os.IsNotExist(err) {
		t.Error("main.go should not be copied")
	}
}

func TestCopyLegalFiles_DoesNotOverwrite(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	os.WriteFile(filepath.Join(srcDir, "LICENSE"), []byte("upstream license"), 0644)
	os.WriteFile(filepath.Join(destDir, "LICENSE"), []byte("existing license"), 0644)

	if err := CopyLegalFiles(srcDir, destDir); err != nil {
		t.Fatalf("CopyLegalFiles: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(destDir, "LICENSE"))
	if string(data) != "existing license" {
		t.Error("existing LICENSE should not be overwritten")
	}
}

func TestCopyLegalFiles_SurvivesExcludePatterns(t *testing.T) {
	// Simulate a git-raw workflow where *.md is excluded but README.md should still appear
	srcDir := t.TempDir()
	destDir := t.TempDir()
	runtimeDir := filepath.Join(destDir, "Runtime")
	os.MkdirAll(runtimeDir, 0755)

	os.WriteFile(filepath.Join(srcDir, "Foo.cs"), []byte("class Foo {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "LICENSE"), []byte("MIT"), 0644)
	os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("readme"), 0644)

	// CopyFiltered with *.md excluded — README.md goes to Runtime but gets excluded
	CopyFiltered(srcDir, runtimeDir, []string{"*.md"})

	// CopyLegalFiles to package root — should still pick up README.md and LICENSE
	if err := CopyLegalFiles(srcDir, destDir); err != nil {
		t.Fatalf("CopyLegalFiles: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "LICENSE")); os.IsNotExist(err) {
		t.Error("LICENSE should be at package root")
	}
	if _, err := os.Stat(filepath.Join(destDir, "README.md")); os.IsNotExist(err) {
		t.Error("README.md should be at package root despite *.md exclude")
	}
}

func TestExtractLegalFilesFromZip(t *testing.T) {
	// Create a zip with legal files at root and in subdirs
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	f, _ := os.Create(zipPath)
	w := zip.NewWriter(f)

	for _, file := range []struct {
		name, content string
	}{
		{"LICENSE.txt", "license text"},
		{"README.md", "readme text"},
		{"lib/netstandard2.0/Foo.dll", "fake dll"},
		{"lib/netstandard2.0/LICENSE", "nested license"},
	} {
		fw, _ := w.Create(file.name)
		fw.Write([]byte(file.content))
	}
	w.Close()
	f.Close()

	destDir := t.TempDir()
	if err := ExtractLegalFilesFromZip(zipPath, destDir); err != nil {
		t.Fatalf("ExtractLegalFilesFromZip: %v", err)
	}

	// Root-level legal files should be extracted
	data, err := os.ReadFile(filepath.Join(destDir, "LICENSE.txt"))
	if err != nil {
		t.Error("LICENSE.txt should be extracted")
	} else if string(data) != "license text" {
		t.Errorf("LICENSE.txt content: got %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(destDir, "README.md"))
	if err != nil {
		t.Error("README.md should be extracted")
	} else if string(data) != "readme text" {
		t.Errorf("README.md content: got %q", string(data))
	}

	// Nested LICENSE should NOT be extracted (not at zip root)
	// Only root-level files in lib/ subdir shouldn't appear at destDir root
	if _, err := os.Stat(filepath.Join(destDir, "Foo.dll")); !os.IsNotExist(err) {
		t.Error("Foo.dll should not be extracted")
	}
}
