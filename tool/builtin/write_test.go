package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAppendChunks(t *testing.T) {
	w := &Write{}
	path := filepath.Join(t.TempDir(), "nested", "out.txt")

	if _, err := w.Execute(context.Background(), "call_1", map[string]any{
		"path":    path,
		"content": "hello",
	}, nil); err != nil {
		t.Fatalf("initial write: %v", err)
	}
	if _, err := w.Execute(context.Background(), "call_2", map[string]any{
		"path":    path,
		"content": " world",
		"append":  true,
	}, nil); err != nil {
		t.Fatalf("append write: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("content = %q, want %q", string(got), "hello world")
	}
}

func TestWriteSchemaLimitsContentChunkSize(t *testing.T) {
	w := &Write{}
	schema := w.Schema()

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong type: %T", schema["properties"])
	}
	content, ok := props["content"].(map[string]any)
	if !ok {
		t.Fatalf("content schema missing or wrong type: %T", props["content"])
	}
	if got, ok := content["maxLength"].(int); !ok || got != 4000 {
		t.Fatalf("content maxLength = %#v, want 4000", content["maxLength"])
	}
	if desc, _ := content["description"].(string); !strings.Contains(desc, "split larger files into chunks") {
		t.Fatalf("content description does not mention chunking: %q", desc)
	}
	if _, ok := props["append"].(map[string]any); !ok {
		t.Fatalf("append schema missing or wrong type: %T", props["append"])
	}
}
