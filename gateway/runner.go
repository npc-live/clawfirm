package gateway

import (
	"context"

	"github.com/ai-gateway/clawfirm/types"
)

// AgentRunner is the interface that Session uses to drive an agent.
type AgentRunner interface {
	PromptMessages(ctx context.Context, msgs []types.Message) error
	Abort()
	WaitForIdle(ctx context.Context) error
	Subscribe(fn func(types.AgentEvent)) func()
	State() types.AgentState
	ClearMessages()
}
