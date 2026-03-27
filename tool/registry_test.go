package tool

import (
	"context"
	"testing"
)

func makeTool(name string) AgentTool {
	return &BaseToolImpl{
		ToolName:        name,
		ToolDescription: "desc of " + name,
		ToolLabel:       name,
		ToolSchema:      map[string]any{},
		ExecuteFn: func(ctx context.Context, id string, params map[string]any, onUpdate func(ToolUpdate)) (ToolResult, error) {
			return ToolResult{}, nil
		},
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(makeTool("echo"))

	got, ok := r.Get("echo")
	if !ok {
		t.Fatal("expected ok=true for registered tool")
	}
	if got.Name() != "echo" {
		t.Errorf("Name: got %q want echo", got.Name())
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for missing tool")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register(makeTool("a"))
	r.Register(makeTool("b"))
	r.Register(makeTool("c"))

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() len: got %d want 3", len(all))
	}
}

func TestRegistryOverwrite(t *testing.T) {
	r := NewRegistry()
	r.Register(makeTool("echo"))
	r.Register(makeTool("echo")) // overwrite

	all := r.All()
	if len(all) != 1 {
		t.Errorf("expected 1 tool after overwrite, got %d", len(all))
	}
}
