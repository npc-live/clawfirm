package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// MockResponse is a canned response for MockProvider.
type MockResponse struct {
	Events []types.AssistantMessageEvent
	Err    error
}

// MockProvider is a test double for provider.LLMProvider.
// It replays pre-registered response sequences.
type MockProvider struct {
	id        string
	models    []types.Model
	responses []MockResponse
	callCount int
	mu        sync.Mutex
}

// NewMockProvider creates a MockProvider with the given ID.
func NewMockProvider(id string) *MockProvider {
	return &MockProvider{id: id}
}

// ID returns the provider identifier.
func (m *MockProvider) ID() string { return m.id }

// Models returns the provider's model list.
func (m *MockProvider) Models() []types.Model {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.models
}

// SetModels sets the model list returned by Models().
func (m *MockProvider) SetModels(models []types.Model) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models = models
}

// AddResponse appends a canned response.
func (m *MockProvider) AddResponse(events ...types.AssistantMessageEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, MockResponse{Events: events})
}

// AddErrorResponse appends a canned error response.
func (m *MockProvider) AddErrorResponse(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, MockResponse{Err: err})
}

// AddTextResponse appends a simple text-reply event sequence.
func (m *MockProvider) AddTextResponse(text string) {
	ts := time.Now().UnixMilli()
	finalMsg := &types.AssistantMessage{
		Role:       "assistant",
		Content:    []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
		StopReason: types.StopReasonStop,
		Timestamp:  ts,
	}
	m.AddResponse(
		types.AssistantMessageEvent{Type: types.StreamEventStart},
		types.AssistantMessageEvent{Type: types.StreamEventTextStart, ContentIndex: 0},
		types.AssistantMessageEvent{Type: types.StreamEventTextDelta, ContentIndex: 0, Delta: text},
		types.AssistantMessageEvent{Type: types.StreamEventTextEnd, ContentIndex: 0, Content: text},
		types.AssistantMessageEvent{Type: types.StreamEventDone, Message: finalMsg, Reason: types.StopReasonStop},
	)
}

// AddToolCallResponse appends a tool-call event sequence.
func (m *MockProvider) AddToolCallResponse(name string, args map[string]any) {
	ts := time.Now().UnixMilli()
	tc := &types.ToolCall{
		Type:      types.ContentTypeToolCall,
		ID:        "call_" + name,
		Name:      name,
		Arguments: args,
	}
	finalMsg := &types.AssistantMessage{
		Role:       "assistant",
		Content:    []types.ContentBlock{tc},
		StopReason: types.StopReasonToolUse,
		Timestamp:  ts,
	}
	m.AddResponse(
		types.AssistantMessageEvent{Type: types.StreamEventStart},
		types.AssistantMessageEvent{Type: types.StreamEventToolCallStart, ContentIndex: 0, ToolCall: tc},
		types.AssistantMessageEvent{Type: types.StreamEventToolCallEnd, ContentIndex: 0, ToolCall: tc},
		types.AssistantMessageEvent{Type: types.StreamEventDone, Message: finalMsg, Reason: types.StopReasonToolUse},
	)
}

// CallCount returns how many times Stream was called.
func (m *MockProvider) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Stream replays the next pre-registered response.
// The LLMRequest parameter type is declared as any to avoid circular imports;
// callers cast it appropriately. The canonical signature matches provider.LLMProvider.
func (m *MockProvider) Stream(ctx context.Context, req any) (<-chan types.AssistantMessageEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if len(m.responses) == 0 {
		return nil, fmt.Errorf("MockProvider: no more responses")
	}
	resp := m.responses[0]
	m.responses = m.responses[1:]

	if resp.Err != nil {
		return nil, resp.Err
	}

	ch := make(chan types.AssistantMessageEvent, len(resp.Events))
	for _, ev := range resp.Events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
