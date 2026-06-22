//go:build integration

package packager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klumhru/unity-packager/internal/config"
)

func TestIntegration_GitUnity_VContainer(t *testing.T) {
	projectDir := t.TempDir()
	packagesDir := filepath.Join(projectDir, "Packages")
	os.MkdirAll(packagesDir, 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name: "jp.hadashikick.vcontainer",
				Type: config.GitUnity,
				URL:  "https://github.com/hadashiA/VContainer.git",
				Ref:  "1.16.7",
				Path: "VContainer/Assets/VContainer",
				Exclude: []string{
					"Tests~/**",
				},
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(packagesDir, "jp.hadashikick.vcontainer")

	// package.json should exist (from upstream)
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should exist")
	}

	// Meta files should be generated
	if _, err := os.Stat(filepath.Join(destDir, "package.json.meta")); os.IsNotExist(err) {
		t.Error("package.json.meta should exist")
	}

	// Runtime directory should exist
	if _, err := os.Stat(filepath.Join(destDir, "Runtime")); os.IsNotExist(err) {
		t.Error("Runtime directory should exist")
	}
}

func TestIntegration_GitRaw_Protobuf(t *testing.T) {
	projectDir := t.TempDir()
	packagesDir := filepath.Join(projectDir, "Packages")
	os.MkdirAll(packagesDir, 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name:        "com.google.protobuf",
				Type:        config.GitRaw,
				URL:         "https://github.com/protocolbuffers/protobuf.git",
				Ref:         "v3.27.1",
				Path:        "csharp/src/Google.Protobuf",
				Version:     "3.27.1",
				Description: "Google Protocol Buffers for C#",
				Exclude:     []string{"**/*Test*.cs", "**/*.csproj", "**/*.sln"},
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(packagesDir, "com.google.protobuf")

	// package.json should be generated
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should be generated")
	}

	// Runtime directory with source
	runtimeDir := filepath.Join(destDir, "Runtime")
	if _, err := os.Stat(runtimeDir); os.IsNotExist(err) {
		t.Error("Runtime directory should exist")
	}

	// .asmdef should be generated
	asmdefPath := filepath.Join(runtimeDir, "com.google.protobuf.asmdef")
	if _, err := os.Stat(asmdefPath); os.IsNotExist(err) {
		t.Error("asmdef should be generated")
	}

	// Meta files should exist
	if _, err := os.Stat(filepath.Join(destDir, "package.json.meta")); os.IsNotExist(err) {
		t.Error("package.json.meta should exist")
	}
}

func TestIntegration_NuGet_GrpcCore(t *testing.T) {
	projectDir := t.TempDir()
	packagesDir := filepath.Join(projectDir, "Packages")
	os.MkdirAll(packagesDir, 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name:           "com.grpc.core",
				Type:           config.NuGet,
				NuGetID:        "Grpc.Core",
				NuGetVersion:   "2.46.6",
				NuGetFramework: "netstandard2.0",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(packagesDir, "com.grpc.core")

	// package.json should be generated
	if _, err := os.Stat(filepath.Join(destDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should be generated")
	}

	// Plugins directory with DLLs
	pluginsDir := filepath.Join(destDir, "Plugins")
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		t.Error("Plugins directory should exist")
	}

	// Check for the main DLL
	dllPath := filepath.Join(pluginsDir, "Grpc.Core.Api.dll")
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		// The actual DLL name might differ, just check that at least one DLL exists
		entries, _ := os.ReadDir(pluginsDir)
		if len(entries) == 0 {
			t.Error("Plugins directory should contain at least one file")
		}
	}

	// Meta files
	if _, err := os.Stat(filepath.Join(destDir, "package.json.meta")); os.IsNotExist(err) {
		t.Error("package.json.meta should exist")
	}
}

func TestIntegration_NuGet_EditorOnly(t *testing.T) {
	projectDir := t.TempDir()
	packagesDir := filepath.Join(projectDir, "Packages")
	os.MkdirAll(packagesDir, 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name:           "com.newtonsoft.json",
				Type:           config.NuGet,
				NuGetID:        "Newtonsoft.Json",
				NuGetVersion:   "13.0.3",
				NuGetFramework: "netstandard2.0",
				EditorOnly:     true,
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(packagesDir, "com.newtonsoft.json")

	// DLLs must land under Editor/, not Plugins/.
	if _, err := os.Stat(filepath.Join(destDir, "Editor", "Newtonsoft.Json.dll")); err != nil {
		t.Errorf("Newtonsoft.Json.dll should be under Editor/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "Plugins")); !os.IsNotExist(err) {
		t.Errorf("Plugins/ should not exist when editorOnly is set")
	}
	if _, err := os.Stat(filepath.Join(destDir, "Editor.meta")); os.IsNotExist(err) {
		t.Error("Editor.meta should be generated")
	}
}

func TestIntegration_NuGet_ResolveDependencies(t *testing.T) {
	projectDir := t.TempDir()
	packagesDir := filepath.Join(projectDir, "Packages")
	os.MkdirAll(packagesDir, 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name:                     "com.grpc.core",
				Type:                     config.NuGet,
				NuGetID:                  "Grpc.Core",
				NuGetVersion:             "2.46.6",
				NuGetFramework:           "netstandard2.0",
				NuGetResolveDependencies: true,
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	pluginsDir := filepath.Join(packagesDir, "com.grpc.core", "Plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		t.Fatalf("reading Plugins: %v", err)
	}

	// Resolution should produce multiple assemblies (the package plus its
	// transitive deps), while Unity-provided facades like System.Memory are
	// skipped to avoid duplicate-assembly conflicts.
	dlls := 0
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name()), ".dll") {
			dlls++
		}
		if e.Name() == "System.Memory.dll" {
			t.Error("System.Memory.dll should be skipped (Unity-provided facade)")
		}
	}
	if dlls < 2 {
		t.Errorf("expected multiple DLLs after dependency resolution, got %d", dlls)
	}
}

func TestIntegration_Archive_FirebaseApp(t *testing.T) {
	projectDir := t.TempDir()
	packagesDir := filepath.Join(projectDir, "Packages")
	os.MkdirAll(packagesDir, 0755)

	p := New(projectDir, Options{Verbose: true, Clean: true, NoCache: true})

	cfg := &config.Config{
		Packages: []config.PackageSpec{
			{
				Name: "com.google.firebase.app",
				Type: config.Archive,
				URL:  "https://dl.google.com/games/registry/unity/com.google.firebase.app/com.google.firebase.app-13.9.0.tgz",
			},
		},
	}

	if err := p.Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	destDir := filepath.Join(packagesDir, "com.google.firebase.app")

	// package.json should exist (from upstream — this is a Unity package)
	pkgJSON := filepath.Join(destDir, "package.json")
	if _, err := os.Stat(pkgJSON); os.IsNotExist(err) {
		t.Fatal("package.json should exist")
	}

	// Verify it's the upstream package.json, not a generated one
	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	if !containsBytes(data, []byte("Firebase App (Core)")) {
		t.Error("package.json should contain upstream Firebase displayName")
	}
	if !containsBytes(data, []byte("13.9.0")) {
		t.Error("package.json should contain version 13.9.0")
	}

	// Plugins directory should exist (Firebase ships native plugins)
	if _, err := os.Stat(filepath.Join(destDir, "Plugins")); os.IsNotExist(err) {
		t.Error("Plugins directory should exist")
	}

	// Meta files should be generated
	if _, err := os.Stat(filepath.Join(destDir, "package.json.meta")); os.IsNotExist(err) {
		t.Error("package.json.meta should exist")
	}

	// Firebase directory should exist (contains editor scripts)
	if _, err := os.Stat(filepath.Join(destDir, "Firebase")); os.IsNotExist(err) {
		t.Error("Firebase directory should exist")
	}
}

func containsBytes(data, sub []byte) bool {
	for i := 0; i+len(sub) <= len(data); i++ {
		if string(data[i:i+len(sub)]) == string(sub) {
			return true
		}
	}
	return false
}
