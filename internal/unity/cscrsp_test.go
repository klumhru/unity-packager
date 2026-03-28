package unity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCscRsp(t *testing.T) {
	dir := t.TempDir()

	if err := WriteCscRsp(dir, []string{"0618", "0649"}); err != nil {
		t.Fatalf("WriteCscRsp: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "csc.rsp"))
	if err != nil {
		t.Fatalf("reading csc.rsp: %v", err)
	}

	expected := "-nowarn:0618,0649\n"
	if string(data) != expected {
		t.Errorf("csc.rsp content: got %q, want %q", string(data), expected)
	}
}

func TestWriteCscRsp_SingleWarning(t *testing.T) {
	dir := t.TempDir()

	if err := WriteCscRsp(dir, []string{"0618"}); err != nil {
		t.Fatalf("WriteCscRsp: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "csc.rsp"))
	expected := "-nowarn:0618\n"
	if string(data) != expected {
		t.Errorf("csc.rsp content: got %q, want %q", string(data), expected)
	}
}

func TestWriteCscRsp_Empty(t *testing.T) {
	dir := t.TempDir()

	if err := WriteCscRsp(dir, nil); err != nil {
		t.Fatalf("WriteCscRsp: %v", err)
	}

	// Should not create file
	if _, err := os.Stat(filepath.Join(dir, "csc.rsp")); !os.IsNotExist(err) {
		t.Error("csc.rsp should not be created when no warnings specified")
	}
}

func TestWriteCscRspForAsmdefs(t *testing.T) {
	dir := t.TempDir()

	// Create multiple asmdef files in different directories
	runtimeDir := filepath.Join(dir, "Runtime")
	editorDir := filepath.Join(dir, "Editor")
	os.MkdirAll(runtimeDir, 0755)
	os.MkdirAll(editorDir, 0755)

	os.WriteFile(filepath.Join(runtimeDir, "com.example.runtime.asmdef"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(editorDir, "com.example.editor.asmdef"), []byte("{}"), 0644)

	if err := WriteCscRspForAsmdefs(dir, []string{"0618"}); err != nil {
		t.Fatalf("WriteCscRspForAsmdefs: %v", err)
	}

	// Both directories should have csc.rsp
	for _, d := range []string{runtimeDir, editorDir} {
		data, err := os.ReadFile(filepath.Join(d, "csc.rsp"))
		if err != nil {
			t.Errorf("csc.rsp not found in %s: %v", d, err)
			continue
		}
		if string(data) != "-nowarn:0618\n" {
			t.Errorf("csc.rsp in %s: got %q", d, string(data))
		}
	}

	// Root dir should NOT have csc.rsp (no asmdef there)
	if _, err := os.Stat(filepath.Join(dir, "csc.rsp")); !os.IsNotExist(err) {
		t.Error("root dir should not have csc.rsp")
	}
}

func TestWriteCscRspForAsmdefs_Empty(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.asmdef"), []byte("{}"), 0644)

	if err := WriteCscRspForAsmdefs(dir, nil); err != nil {
		t.Fatalf("WriteCscRspForAsmdefs: %v", err)
	}

	// No csc.rsp should be created
	if _, err := os.Stat(filepath.Join(dir, "csc.rsp")); !os.IsNotExist(err) {
		t.Error("csc.rsp should not be created when no warnings specified")
	}
}
