package types

// AgentEventType identifies what happened in the agent lifecycle.
type AgentEventType string

const (
	EventAgentStart          AgentEventType = "agent_start"
	EventAgentEnd            AgentEventType = "agent_end"
	EventTurnStart           AgentEventType = "turn_start"
	EventTurnEnd             AgentEventType = "turn_end"
	EventMessageStart        AgentEventType = "message_start"
	EventMessageUpdate       AgentEventType = "message_update"
	EventMessageEnd          AgentEventType = "message_end"
	EventToolExecutionStart  AgentEventType = "tool_execution_start"
	EventToolExecutionUpdate AgentEventType = "tool_execution_update"
	EventToolExecutionEnd    AgentEventType = "tool_execution_end"
)

// AgentEvent carries information about a single step in the agent lifecycle.
// A struct with a Type field is used instead of an interface to keep event handling simple.
type AgentEvent struct {
	Type AgentEventType `json:"type"`

	// agent_end: accumulated messages for the full run
	Messages []Message `json:"messages,omitempty"`

	// turn_end: final assistant message and any tool results for the turn
	Message     Message             `json:"message,omitempty"`
	ToolResults []ToolResultMessage `json:"toolResults,omitempty"`

	// message_start / message_update / message_end
	AssistantMsg *AssistantMessage      `json:"assistantMessage,omitempty"`
	StreamEvent  *AssistantMessageEvent `json:"streamEvent,omitempty"`

	// tool_execution_*
	ToolCallID    string `json:"toolCallId,omitempty"`
	ToolName      string `json:"toolName,omitempty"`
	ToolArgs      any    `json:"toolArgs,omitempty"`
	ToolResult    any    `json:"toolResult,omitempty"`
	ToolIsError   bool   `json:"toolIsError,omitempty"`
	PartialResult any    `json:"partialResult,omitempty"`
}
