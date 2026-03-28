package unity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferRootNamespace(t *testing.T) {
	dir := t.TempDir()

	// Create files with various namespaces, simulating Google.Protobuf structure
	files := map[string]string{
		"MessageParser.cs":                     "namespace Google.Protobuf\n{\n    public class MessageParser {}\n}",
		"ByteString.cs":                        "namespace Google.Protobuf\n{\n    public class ByteString {}\n}",
		"Collections/RepeatedField.cs":         "namespace Google.Protobuf.Collections\n{\n    public class RepeatedField {}\n}",
		"Collections/MapField.cs":              "namespace Google.Protobuf.Collections\n{\n    public class MapField {}\n}",
		"Reflection/MessageDescriptor.cs":      "namespace Google.Protobuf.Reflection\n{\n    public class MessageDescriptor {}\n}",
		"WellKnownTypes/Timestamp.cs":          "namespace Google.Protobuf.WellKnownTypes\n{\n    public class Timestamp {}\n}",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	ns := InferRootNamespace(dir)
	if ns != "Google.Protobuf" {
		t.Errorf("InferRootNamespace: got %q, want %q", ns, "Google.Protobuf")
	}
}

func TestInferRootNamespace_Empty(t *testing.T) {
	dir := t.TempDir()
	ns := InferRootNamespace(dir)
	if ns != "" {
		t.Errorf("expected empty namespace for empty dir, got %q", ns)
	}
}

func TestInferRootNamespace_SingleNamespace(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Foo.cs"), []byte("namespace MyLib\n{\n}"), 0644)
	os.WriteFile(filepath.Join(dir, "Bar.cs"), []byte("namespace MyLib\n{\n}"), 0644)

	ns := InferRootNamespace(dir)
	if ns != "MyLib" {
		t.Errorf("InferRootNamespace: got %q, want %q", ns, "MyLib")
	}
}

func TestExtractNamespace(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		content string
		want    string
	}{
		{"namespace Foo.Bar\n{", "Foo.Bar"},
		{"  namespace Foo.Bar\n{", "Foo.Bar"},
		{"namespace Foo.Bar;", "Foo.Bar"},
		{"// comment\nnamespace Foo\n{", "Foo"},
		{"using System;\nnamespace Foo\n{", "Foo"},
		{"// no namespace here", ""},
	}

	for i, tt := range tests {
		path := filepath.Join(dir, "test"+string(rune('0'+i))+".cs")
		os.WriteFile(path, []byte(tt.content), 0644)
		got := extractNamespace(path)
		if got != tt.want {
			t.Errorf("extractNamespace(%q) = %q, want %q", tt.content, got, tt.want)
		}
	}
}
