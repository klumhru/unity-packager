package packager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"Tests~/Foo.cs", []string{"Tests~/**"}, true},
		{"Tests~/Sub/Bar.cs", []string{"Tests~/**"}, true},
		{"Runtime/Foo.cs", []string{"Tests~/**"}, false},
		{"Foo_test.cs", []string{"**/*_test.cs"}, true},
		{"Sub/Foo_test.cs", []string{"**/*_test.cs"}, true},
		{"Sub/Foo.cs", []string{"**/*_test.cs"}, false},
		{"README.md", []string{"*.md"}, true},
		{"docs/README.md", []string{"*.md"}, false},       // *.md only matches at root
		{"docs/README.md", []string{"**/*.md"}, true},      // **/*.md matches anywhere
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ShouldExclude(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("ShouldExclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestCopyFiltered(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create source structure
	os.MkdirAll(filepath.Join(srcDir, "Runtime"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "Tests~"), 0755)
	os.WriteFile(filepath.Join(srcDir, "Runtime", "Foo.cs"), []byte("class Foo {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "Tests~", "FooTest.cs"), []byte("class FooTest {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# readme"), 0644)

	err := CopyFiltered(srcDir, destDir, []string{"*.md"})
	if err != nil {
		t.Fatalf("CopyFiltered: %v", err)
	}

	// Foo.cs should be copied
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Foo.cs")); os.IsNotExist(err) {
		t.Error("Runtime/Foo.cs should be copied")
	}

	// Tests~ should be excluded automatically (tilde dir)
	if _, err := os.Stat(filepath.Join(destDir, "Tests~", "FooTest.cs")); !os.IsNotExist(err) {
		t.Error("Tests~/FooTest.cs should be excluded (tilde directory)")
	}

	// README.md should be excluded by pattern
	if _, err := os.Stat(filepath.Join(destDir, "README.md")); !os.IsNotExist(err) {
		t.Error("README.md should be excluded")
	}
}

func TestCopyFiltered_SkipsTildeDirs(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create various tilde directories
	os.MkdirAll(filepath.Join(srcDir, "Tests~"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "Documentation~"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "Samples~"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "Runtime"), 0755)
	os.WriteFile(filepath.Join(srcDir, "Tests~", "Test.cs"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(srcDir, "Documentation~", "index.md"), []byte("docs"), 0644)
	os.WriteFile(filepath.Join(srcDir, "Samples~", "Example.cs"), []byte("sample"), 0644)
	os.WriteFile(filepath.Join(srcDir, "Runtime", "Lib.cs"), []byte("lib"), 0644)

	// No exclude patterns — tilde dirs should still be skipped
	err := CopyFiltered(srcDir, destDir, nil)
	if err != nil {
		t.Fatalf("CopyFiltered: %v", err)
	}

	// Runtime should be copied
	if _, err := os.Stat(filepath.Join(destDir, "Runtime", "Lib.cs")); os.IsNotExist(err) {
		t.Error("Runtime/Lib.cs should be copied")
	}

	// All tilde dirs should be skipped
	for _, dir := range []string{"Tests~", "Documentation~", "Samples~"} {
		if _, err := os.Stat(filepath.Join(destDir, dir)); !os.IsNotExist(err) {
			t.Errorf("%s should not be copied (Unity-ignored tilde directory)", dir)
		}
	}
}

func TestCopyFiltered_SkipsGitDir(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	os.MkdirAll(filepath.Join(srcDir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(srcDir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644)
	os.WriteFile(filepath.Join(srcDir, "Foo.cs"), []byte("class Foo {}"), 0644)

	err := CopyFiltered(srcDir, destDir, nil)
	if err != nil {
		t.Fatalf("CopyFiltered: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, ".git")); !os.IsNotExist(err) {
		t.Error(".git directory should not be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "Foo.cs")); os.IsNotExist(err) {
		t.Error("Foo.cs should be copied")
	}
}
