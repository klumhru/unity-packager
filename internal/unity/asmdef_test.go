package unity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAsmDef(t *testing.T) {
	asmdef := NewAsmDef("com.example.test", "Example.Test", []string{"com.example.dep"})

	if asmdef.Name != "com.example.test" {
		t.Errorf("Name: got %q", asmdef.Name)
	}
	if asmdef.RootNamespace != "Example.Test" {
		t.Errorf("RootNamespace: got %q", asmdef.RootNamespace)
	}
	if len(asmdef.References) != 1 || asmdef.References[0] != "com.example.dep" {
		t.Errorf("References: got %v", asmdef.References)
	}
	if !asmdef.AutoReferenced {
		t.Error("AutoReferenced should be true")
	}
}

func TestWriteAsmDef(t *testing.T) {
	dir := t.TempDir()
	asmdef := NewAsmDef("com.example.test", "Example.Test", nil)

	if err := WriteAsmDef(dir, "com.example.test.asmdef", asmdef); err != nil {
		t.Fatalf("WriteAsmDef: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "com.example.test.asmdef"))
	if err != nil {
		t.Fatalf("reading asmdef: %v", err)
	}

	var parsed AsmDef
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parsing asmdef: %v", err)
	}

	if parsed.Name != "com.example.test" {
		t.Errorf("parsed name: got %q", parsed.Name)
	}
	if parsed.RootNamespace != "Example.Test" {
		t.Errorf("parsed rootNamespace: got %q", parsed.RootNamespace)
	}
}
