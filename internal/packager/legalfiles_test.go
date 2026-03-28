package packager

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestIsLegalFile(t *testing.T) {
	yes := []string{
		"LICENSE", "LICENSE.md", "LICENSE.txt", "LICENSE.rst",
		"license", "License.md",
		"LICENCE", "Licence.txt", "LICENCE.rst",
		"README", "README.md", "Readme.txt", "README.rst",
		"NOTICE", "NOTICE.md", "NOTICE.txt", "NOTICE.rst",
		"ThirdPartyNotices.txt", "ThirdPartyNotices.md", "ThirdPartyNotices.rst",
		"third-party-notices", "THIRD-PARTY-NOTICES.md",
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
	if err := CopyFiltered(srcDir, runtimeDir, []string{"*.md"}); err != nil {
		t.Fatalf("CopyFiltered: %v", err)
	}

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

func TestCopyLegalFilesSearchingUp(t *testing.T) {
	// Simulate a repo with LICENSE at root and source in a subpath
	repoRoot := t.TempDir()
	subPath := filepath.Join(repoRoot, "csharp", "src", "MyLib")
	os.MkdirAll(subPath, 0755)

	// License at repo root only
	os.WriteFile(filepath.Join(repoRoot, "LICENSE"), []byte("Apache 2.0"), 0644)
	// README at subpath level
	os.WriteFile(filepath.Join(subPath, "README.md"), []byte("lib readme"), 0644)
	// Source file
	os.WriteFile(filepath.Join(subPath, "Foo.cs"), []byte("class Foo {}"), 0644)

	destDir := t.TempDir()
	if err := CopyLegalFilesSearchingUp(subPath, repoRoot, destDir); err != nil {
		t.Fatalf("CopyLegalFilesSearchingUp: %v", err)
	}

	// LICENSE should be found from repo root
	data, err := os.ReadFile(filepath.Join(destDir, "LICENSE"))
	if err != nil {
		t.Fatal("LICENSE should be copied from repo root")
	}
	if string(data) != "Apache 2.0" {
		t.Errorf("LICENSE content: got %q", string(data))
	}

	// README.md should be found from subpath (closer takes precedence)
	data, err = os.ReadFile(filepath.Join(destDir, "README.md"))
	if err != nil {
		t.Fatal("README.md should be copied from subpath")
	}
	if string(data) != "lib readme" {
		t.Errorf("README.md content: got %q", string(data))
	}
}

func TestCopyLegalFilesSearchingUp_CloserTakesPrecedence(t *testing.T) {
	repoRoot := t.TempDir()
	subPath := filepath.Join(repoRoot, "src")
	os.MkdirAll(subPath, 0755)

	// LICENSE at both levels
	os.WriteFile(filepath.Join(repoRoot, "LICENSE"), []byte("root license"), 0644)
	os.WriteFile(filepath.Join(subPath, "LICENSE"), []byte("sub license"), 0644)

	destDir := t.TempDir()
	if err := CopyLegalFilesSearchingUp(subPath, repoRoot, destDir); err != nil {
		t.Fatalf("CopyLegalFilesSearchingUp: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(destDir, "LICENSE"))
	if string(data) != "sub license" {
		t.Errorf("closer LICENSE should take precedence, got %q", string(data))
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

	// Nested files should NOT be extracted (not at zip root)
	if _, err := os.Stat(filepath.Join(destDir, "Foo.dll")); !os.IsNotExist(err) {
		t.Error("Foo.dll should not be extracted")
	}

	// Nested LICENSE (under lib/netstandard2.0/) should not be extracted
	// Verify only the root-level files exist by checking total file count
	entries, err := os.ReadDir(destDir)
	if err != nil {
		t.Fatalf("failed to read destination directory: %v", err)
	}
	if len(entries) != 2 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected exactly 2 files (LICENSE.txt, README.md), got %d: %v", len(entries), names)
	}
}
