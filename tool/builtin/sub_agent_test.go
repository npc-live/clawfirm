package builtin

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestSubAgent_MissingPrompt(t *testing.T) {
	sa := &SubAgent{}
	_, err := sa.Execute(context.Background(), "1", map[string]any{"description": "test"}, nopUpdate)
	if err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Errorf("expected prompt-required error, got %v", err)
	}
}

func TestSubAgent_NilSpawnFn(t *testing.T) {
	sa := &SubAgent{}
	res, err := sa.Execute(context.Background(), "1", map[string]any{
		"prompt": "do something",
	}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "not available") {
		t.Errorf("expected not-available message, got %q", text)
	}
}

func TestSubAgent_Success(t *testing.T) {
	sa := &SubAgent{
		SpawnFn: func(ctx context.Context, req SubAgentRequest) (SubAgentResult, error) {
			if req.Description != "find bugs" {
				t.Errorf("description: got %q", req.Description)
			}
			if req.Prompt != "Search for bugs in main.go" {
				t.Errorf("prompt: got %q", req.Prompt)
			}
			if req.Model != "fast-model" {
				t.Errorf("model: got %q", req.Model)
			}
			return SubAgentResult{Output: "Found 3 bugs in main.go"}, nil
		},
	}
	res, err := sa.Execute(context.Background(), "1", map[string]any{
		"description": "find bugs",
		"prompt":      "Search for bugs in main.go",
		"model":       "fast-model",
	}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "Found 3 bugs") {
		t.Errorf("expected sub-agent output, got %q", text)
	}
}

func TestSubAgent_SpawnError(t *testing.T) {
	sa := &SubAgent{
		SpawnFn: func(ctx context.Context, req SubAgentRequest) (SubAgentResult, error) {
			return SubAgentResult{}, fmt.Errorf("spawn failed: resource exhausted")
		},
	}
	res, err := sa.Execute(context.Background(), "1", map[string]any{
		"prompt": "do work",
	}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "Sub-agent failed") {
		t.Errorf("expected failure message, got %q", text)
	}
}

func TestSubAgent_PartialOutputWithError(t *testing.T) {
	sa := &SubAgent{
		SpawnFn: func(ctx context.Context, req SubAgentRequest) (SubAgentResult, error) {
			return SubAgentResult{
				Output: "Partial results here",
				Error:  "timeout after 30s",
			}, nil
		},
	}
	res, err := sa.Execute(context.Background(), "1", map[string]any{
		"prompt": "long task",
	}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "timeout after 30s") {
		t.Errorf("expected error in output, got %q", text)
	}
	if !strings.Contains(text, "Partial results here") {
		t.Errorf("expected partial output, got %q", text)
	}
}

func TestSubAgent_DefaultDescription(t *testing.T) {
	sa := &SubAgent{
		SpawnFn: func(ctx context.Context, req SubAgentRequest) (SubAgentResult, error) {
			if req.Description != "sub-agent task" {
				t.Errorf("expected default description, got %q", req.Description)
			}
			return SubAgentResult{Output: "done"}, nil
		},
	}
	_, err := sa.Execute(context.Background(), "1", map[string]any{
		"prompt": "do something",
	}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubAgent_Schema(t *testing.T) {
	sa := &SubAgent{}
	s := sa.Schema()
	props := s["properties"].(map[string]any)
	if _, ok := props["prompt"]; !ok {
		t.Error("schema missing prompt property")
	}
	if _, ok := props["description"]; !ok {
		t.Error("schema missing description property")
	}
	if _, ok := props["model"]; !ok {
		t.Error("schema missing model property")
	}
	req := s["required"].([]string)
	if len(req) != 2 {
		t.Errorf("required: got %v want [description, prompt]", req)
	}
}

func TestSubAgent_NameAndLabel(t *testing.T) {
	sa := &SubAgent{}
	if sa.Name() != "sub_agent" {
		t.Errorf("Name: got %q", sa.Name())
	}
	if sa.Label() == "" {
		t.Error("Label should not be empty")
	}
}
