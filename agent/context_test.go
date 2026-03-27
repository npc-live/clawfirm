package agent

import (
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

func TestAgentContextClone(t *testing.T) {
	orig := AgentContext{
		SystemPrompt: "you are a helper",
		Messages: []types.Message{
			&types.UserMessage{Role: "user"},
		},
	}

	clone := orig.Clone()
	if clone.SystemPrompt != orig.SystemPrompt {
		t.Errorf("SystemPrompt: got %q want %q", clone.SystemPrompt, orig.SystemPrompt)
	}
	if len(clone.Messages) != len(orig.Messages) {
		t.Errorf("Messages len: got %d want %d", len(clone.Messages), len(orig.Messages))
	}

	// Mutating clone should not affect original
	clone.Messages = append(clone.Messages, &types.UserMessage{Role: "user"})
	if len(orig.Messages) != 1 {
		t.Errorf("Original messages mutated after clone append")
	}
}
