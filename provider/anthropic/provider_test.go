package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
)

func testModel() types.Model {
	return types.Model{
		ID:        "claude-sonnet-4-6",
		Provider:  "anthropic",
		MaxTokens: 1024,
	}
}

func makeRequest(model types.Model) provider.LLMRequest {
	return provider.LLMRequest{
		Model:        model,
		SystemPrompt: "You are a test assistant.",
		Messages: []types.Message{
			&types.UserMessage{
				Role:    "user",
				Content: []types.ContentBlock{&types.TextContent{Type: "text", Text: "hello"}},
			},
		},
	}
}

// buildTextSSE returns an SSE stream that produces a simple text response.
func buildTextSSE(text string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`, text)
}

// buildToolCallSSE returns an SSE stream for a tool call response.
func buildToolCallSSE(toolID, toolName, argsJSON string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":20,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":%q,"name":%q}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":%q}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":10}}

event: message_stop
data: {"type":"message_stop"}

`, toolID, toolName, argsJSON)
}

func TestAnthropicTextResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildTextSSE("Hello, world!"))
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	ch, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	var textDeltas []string
	var doneMsg *types.AssistantMessage
	for ev := range ch {
		switch ev.Type {
		case types.StreamEventTextDelta:
			textDeltas = append(textDeltas, ev.Delta)
		case types.StreamEventDone:
			doneMsg = ev.Message
		}
	}

	if len(textDeltas) == 0 {
		t.Error("expected text delta events")
	}
	fullText := ""
	for _, d := range textDeltas {
		fullText += d
	}
	if fullText != "Hello, world!" {
		t.Errorf("text: got %q want %q", fullText, "Hello, world!")
	}
	if doneMsg == nil {
		t.Fatal("expected done event with message")
	}
	if doneMsg.StopReason != types.StopReasonStop {
		t.Errorf("StopReason: got %q want stop", doneMsg.StopReason)
	}
	if doneMsg.Usage.Input != 10 {
		t.Errorf("Usage.Input: got %d want 10", doneMsg.Usage.Input)
	}
}

func TestAnthropicToolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildToolCallSSE("call_abc", "search", `{"query":"go testing"}`))
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	ch, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	var toolCallEnd *types.AssistantMessageEvent
	var doneMsg *types.AssistantMessage
	for ev := range ch {
		evCopy := ev
		switch ev.Type {
		case types.StreamEventToolCallEnd:
			toolCallEnd = &evCopy
		case types.StreamEventDone:
			doneMsg = ev.Message
		}
	}

	if toolCallEnd == nil {
		t.Fatal("expected toolcall_end event")
	}
	if toolCallEnd.ToolCall == nil {
		t.Fatal("toolcall_end ToolCall is nil")
	}
	if toolCallEnd.ToolCall.Name != "search" {
		t.Errorf("ToolCall.Name: got %q want search", toolCallEnd.ToolCall.Name)
	}
	q, _ := toolCallEnd.ToolCall.Arguments["query"].(string)
	if q != "go testing" {
		t.Errorf("ToolCall.Arguments.query: got %q want 'go testing'", q)
	}
	if doneMsg == nil {
		t.Fatal("expected done message")
	}
	if doneMsg.StopReason != types.StopReasonToolUse {
		t.Errorf("StopReason: got %q want toolUse", doneMsg.StopReason)
	}
}

func TestAnthropicAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"type":"authentication_error","message":"Invalid API key"}}`)
	}))
	defer srv.Close()

	p := NewWithBaseURL("bad-key", srv.URL)
	_, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestAnthropicCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Write partial SSE then block
		fmt.Fprint(w, "event: message_start\ndata: {}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// block until client disconnects
		<-r.Context().Done()
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := p.Stream(ctx, makeRequest(testModel()))
	if err != nil {
		t.Fatalf("unexpected Stream error: %v", err)
	}

	// Read one event to confirm stream started
	<-ch
	cancel()

	// Drain channel; it must close
	for range ch {
	}
}

func TestAnthropicModels(t *testing.T) {
	p := New("key")
	models := p.Models()
	if len(models) == 0 {
		t.Error("expected non-empty models list")
	}
	for _, m := range models {
		if m.Provider != "anthropic" {
			t.Errorf("model %q: Provider = %q, want anthropic", m.ID, m.Provider)
		}
	}
}
