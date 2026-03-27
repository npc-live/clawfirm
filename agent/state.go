package agent

import (
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// AgentState is a read-only snapshot of the agent's current state.
type AgentState struct {
	SystemPrompt     string
	Model            types.Model
	ThinkingLevel    types.ThinkingLevel
	Tools            []tool.AgentTool
	Messages         []types.Message
	IsRunning        bool
	CurrentMsg       *types.AssistantMessage
	PendingToolCalls map[string]bool
	Error            string
}
