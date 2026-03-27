package agent

import (
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

// AgentContext holds the mutable conversation state passed through the agent loop.
type AgentContext struct {
	// SystemPrompt is the system/instructions message sent to the LLM.
	SystemPrompt string
	// Messages is the full conversation history, mutated by the loop.
	Messages []types.Message
	// Tools is the set of tools available for execution.
	Tools []tool.AgentTool
}

// Clone returns a shallow copy of the context.
func (c *AgentContext) Clone() AgentContext {
	msgs := make([]types.Message, len(c.Messages))
	copy(msgs, c.Messages)
	tools := make([]tool.AgentTool, len(c.Tools))
	copy(tools, c.Tools)
	return AgentContext{
		SystemPrompt: c.SystemPrompt,
		Messages:     msgs,
		Tools:        tools,
	}
}
