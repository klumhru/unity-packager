package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "upstream-packages.json")
	data := []byte(`{
		"packages": [
			{
				"name": "com.example.unity-lib",
				"type": "git-unity",
				"url": "https://github.com/org/repo.git",
				"ref": "v1.0.0"
			},
			{
				"name": "com.example.raw-lib",
				"type": "git-raw",
				"url": "https://github.com/org/raw.git",
				"ref": "main",
				"path": "src",
				"version": "1.0.0",
				"dependencies": ["com.example.unity-lib"]
			},
			{
				"name": "com.example.nuget",
				"type": "nuget",
				"nugetId": "Example.Lib",
				"nugetVersion": "2.0.0"
			}
		]
	}`)
	os.WriteFile(cfgPath, data, 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Packages) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(cfg.Packages))
	}

	if cfg.Packages[0].Type != GitUnity {
		t.Errorf("expected type git-unity, got %s", cfg.Packages[0].Type)
	}
	if cfg.Packages[1].Type != GitRaw {
		t.Errorf("expected type git-raw, got %s", cfg.Packages[1].Type)
	}
	if cfg.Packages[2].Type != NuGet {
		t.Errorf("expected type nuget, got %s", cfg.Packages[2].Type)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "empty packages",
			config:  Config{Packages: []PackageSpec{}},
			wantErr: "no packages defined",
		},
		{
			name: "missing name",
			config: Config{Packages: []PackageSpec{
				{Type: GitUnity, URL: "u", Ref: "r"},
			}},
			wantErr: "name is required",
		},
		{
			name: "duplicate name",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: GitUnity, URL: "u", Ref: "r"},
				{Name: "a", Type: GitUnity, URL: "u", Ref: "r"},
			}},
			wantErr: "duplicate name",
		},
		{
			name: "git missing url",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: GitUnity, Ref: "r"},
			}},
			wantErr: "url is required",
		},
		{
			name: "git missing ref",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: GitRaw, URL: "u"},
			}},
			wantErr: "ref is required",
		},
		{
			name: "nuget missing id",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: NuGet, NuGetVersion: "1.0"},
			}},
			wantErr: "nugetId is required",
		},
		{
			name: "nuget missing version",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: NuGet, NuGetID: "Foo"},
			}},
			wantErr: "nugetVersion is required",
		},
		{
			name: "unknown type",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: "bogus"},
			}},
			wantErr: "unknown type",
		},
		{
			name: "unknown dependency",
			config: Config{Packages: []PackageSpec{
				{Name: "a", Type: GitUnity, URL: "u", Ref: "r", Dependencies: []string{"nonexistent"}},
			}},
			wantErr: "dependency \"nonexistent\" not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error %q does not contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	os.WriteFile(cfgPath, []byte(`{invalid`), 0644)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
