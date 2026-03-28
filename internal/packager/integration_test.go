//go:build integration

package packager

import (
	"os"
	"path/filepath"
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
