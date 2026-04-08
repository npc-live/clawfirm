package gateway

import (
	"context"

	"github.com/ai-gateway/clawfirm/types"
)

// AgentRunner is the interface that Session uses to drive an agent.
type AgentRunner interface {
	PromptMessages(ctx context.Context, msgs []types.Message) error
	// ExecuteToolDirectly runs a tool call outside of the LLM loop and injects
	// a synthetic AssistantMessage + ToolResultMessage into the conversation
	// history. Events are emitted through the normal subscription path so the
	// call is persisted and visible to future LLM turns.
	ExecuteToolDirectly(ctx context.Context, toolID, toolName string, args map[string]any, onUpdate func(types.AgentEvent)) error
	Abort()
	WaitForIdle(ctx context.Context) error
	Subscribe(fn func(types.AgentEvent)) func()
	State() types.AgentState
	ClearMessages()
}
