package types

import (
	"encoding/json"
	"testing"
)

func TestModelRoundTrip(t *testing.T) {
	orig := Model{
		ID:       "claude-opus-4-6",
		Name:     "Claude Opus 4.6",
		Provider: "anthropic",
		BaseURL:  "https://api.anthropic.com",
		Reasoning: true,
		InputTypes: []string{"text", "image"},
		Cost: ModelCost{
			Input:  3.0,
			Output: 15.0,
		},
		ContextWindow: 200000,
		MaxTokens:     8192,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Model
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != orig.ID {
		t.Errorf("ID: got %q want %q", got.ID, orig.ID)
	}
	if got.Provider != orig.Provider {
		t.Errorf("Provider: got %q want %q", got.Provider, orig.Provider)
	}
	if !got.Reasoning {
		t.Errorf("Reasoning: expected true")
	}
	if len(got.InputTypes) != 2 {
		t.Errorf("InputTypes len: got %d want 2", len(got.InputTypes))
	}
	if got.Cost.Input != 3.0 {
		t.Errorf("Cost.Input: got %v want 3.0", got.Cost.Input)
	}
	if got.ContextWindow != 200000 {
		t.Errorf("ContextWindow: got %d want 200000", got.ContextWindow)
	}
}

func TestThinkingLevelConstants(t *testing.T) {
	levels := []ThinkingLevel{
		ThinkingLevelOff,
		ThinkingLevelMinimal,
		ThinkingLevelLow,
		ThinkingLevelMedium,
		ThinkingLevelHigh,
		ThinkingLevelXHigh,
	}
	expected := []string{"off", "minimal", "low", "medium", "high", "xhigh"}
	for i, l := range levels {
		if string(l) != expected[i] {
			t.Errorf("ThinkingLevel[%d]: got %q want %q", i, l, expected[i])
		}
	}
}

func TestToolExecutionModeConstants(t *testing.T) {
	if ToolExecutionSequential != "sequential" {
		t.Errorf("ToolExecutionSequential: got %q want sequential", ToolExecutionSequential)
	}
	if ToolExecutionParallel != "parallel" {
		t.Errorf("ToolExecutionParallel: got %q want parallel", ToolExecutionParallel)
	}
}
