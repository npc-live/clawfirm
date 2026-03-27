package types

// AssistantMessageEventType identifies the kind of streaming event emitted during LLM generation.
type AssistantMessageEventType string

const (
	StreamEventStart         AssistantMessageEventType = "start"
	StreamEventTextStart     AssistantMessageEventType = "text_start"
	StreamEventTextDelta     AssistantMessageEventType = "text_delta"
	StreamEventTextEnd       AssistantMessageEventType = "text_end"
	StreamEventThinkingStart AssistantMessageEventType = "thinking_start"
	StreamEventThinkingDelta AssistantMessageEventType = "thinking_delta"
	StreamEventThinkingEnd   AssistantMessageEventType = "thinking_end"
	StreamEventToolCallStart AssistantMessageEventType = "toolcall_start"
	StreamEventToolCallDelta AssistantMessageEventType = "toolcall_delta"
	StreamEventToolCallEnd   AssistantMessageEventType = "toolcall_end"
	StreamEventDone          AssistantMessageEventType = "done"
	StreamEventError         AssistantMessageEventType = "error"
)

// AssistantMessageEvent is a streaming delta event emitted by an LLM provider.
type AssistantMessageEvent struct {
	Type         AssistantMessageEventType `json:"type"`
	ContentIndex int                       `json:"contentIndex,omitempty"`
	Delta        string                    `json:"delta,omitempty"`
	Content      string                    `json:"content,omitempty"`
	ToolCall     *ToolCall                 `json:"toolCall,omitempty"`
	Partial      *AssistantMessage         `json:"partial,omitempty"`
	Message      *AssistantMessage         `json:"message,omitempty"`
	Error        *AssistantMessage         `json:"error,omitempty"`
	Reason       StopReason                `json:"reason,omitempty"`
}

// StreamOptions configures optional parameters for a streaming LLM request.
type StreamOptions struct {
	Temperature   *float64
	MaxTokens     *int
	APIKey        string
	ThinkingLevel ThinkingLevel
	Headers       map[string]string
	SessionID     string
}
