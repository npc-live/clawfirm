package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/types"
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
	// The SDK defers the HTTP request to the goroutine, so Stream() returns a channel.
	// The 401 error surfaces as a StreamEventError on the channel.
	ch, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err != nil {
		return // early error is also acceptable
	}
	var gotError bool
	for ev := range ch {
		if ev.Type == types.StreamEventError {
			gotError = true
		}
	}
	if !gotError {
		t.Fatal("expected error event for 401 response")
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

func TestConvertBlocks_Thinking(t *testing.T) {
	t.Run("thinking with signature keeps signature only", func(t *testing.T) {
		blocks := []types.ContentBlock{
			&types.TextContent{Type: "text", Text: "hello"},
			&types.ThinkingContent{
				Type:              "thinking",
				Thinking:          "long chain of thought... 5000 tokens worth",
				ThinkingSignature: "sig_abc123",
			},
		}
		out, err := convertBlocks(blocks)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(out))
		}
		thinkBlock := out[1]
		if thinkBlock.OfThinking == nil {
			t.Fatal("expected thinking block at index 1")
		}
		if thinkBlock.OfThinking.Thinking != "" {
			t.Errorf("thinking text should be empty, got %q", thinkBlock.OfThinking.Thinking)
		}
		if thinkBlock.OfThinking.Signature != "sig_abc123" {
			t.Errorf("signature should be preserved, got %q", thinkBlock.OfThinking.Signature)
		}
	})

	t.Run("thinking without signature is skipped", func(t *testing.T) {
		blocks := []types.ContentBlock{
			&types.TextContent{Type: "text", Text: "hello"},
			&types.ThinkingContent{
				Type:     "thinking",
				Thinking: "some thinking without signature",
			},
		}
		out, err := convertBlocks(blocks)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 block (thinking skipped), got %d", len(out))
		}
		if out[0].OfText == nil {
			t.Error("remaining block should be text")
		}
	})
}

func TestInjectCacheBreakpoints(t *testing.T) {
	hasCacheControl := func(block anthropicsdk.ContentBlockParamUnion) bool {
		b, _ := json.Marshal(block)
		return strings.Contains(string(b), "cache_control")
	}

	t.Run("no breakpoint for single user message", func(t *testing.T) {
		msgs := []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock("hello")),
		}
		injectCacheBreakpoints(msgs)
		if hasCacheControl(msgs[0].Content[0]) {
			t.Error("single user message should not have cache_control")
		}
	})

	t.Run("breakpoint on second-to-last user message", func(t *testing.T) {
		msgs := []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock("first user msg")),
			anthropicsdk.NewAssistantMessage(anthropicsdk.NewTextBlock("reply")),
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock("second user msg")),
		}
		injectCacheBreakpoints(msgs)

		if !hasCacheControl(msgs[0].Content[0]) {
			b, _ := json.Marshal(msgs[0].Content[0])
			t.Errorf("second-to-last user message should have cache_control, got: %s", b)
		}
		if hasCacheControl(msgs[2].Content[0]) {
			t.Error("last user message should NOT have cache_control")
		}
	})

	t.Run("breakpoint on last block of multi-block message", func(t *testing.T) {
		msgs := []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(
				anthropicsdk.NewTextBlock("block1"),
				anthropicsdk.NewTextBlock("block2"),
			),
			anthropicsdk.NewAssistantMessage(anthropicsdk.NewTextBlock("reply")),
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock("latest")),
		}
		injectCacheBreakpoints(msgs)

		if hasCacheControl(msgs[0].Content[0]) {
			t.Error("first block should not have cache_control")
		}
		if !hasCacheControl(msgs[0].Content[1]) {
			b, _ := json.Marshal(msgs[0].Content[1])
			t.Errorf("last block of target message should have cache_control, got: %s", b)
		}
	})
}

func TestCacheTokenParsing(t *testing.T) {
	ssePayload := `event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":100,"output_tokens":0,"cache_creation_input_tokens":500,"cache_read_input_tokens":2000}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"cached!"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, ssePayload)
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	ch, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	var doneMsg *types.AssistantMessage
	for ev := range ch {
		if ev.Type == types.StreamEventDone {
			doneMsg = ev.Message
		}
	}

	if doneMsg == nil {
		t.Fatal("expected done message")
	}
	if doneMsg.Usage.CacheRead != 2000 {
		t.Errorf("CacheRead: got %d want 2000", doneMsg.Usage.CacheRead)
	}
	if doneMsg.Usage.CacheWrite != 500 {
		t.Errorf("CacheWrite: got %d want 500", doneMsg.Usage.CacheWrite)
	}
	if doneMsg.Usage.Input != 100 {
		t.Errorf("Input: got %d want 100", doneMsg.Usage.Input)
	}
}

func TestRequestIncludesCacheControl(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildTextSSE("ok"))
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	req := provider.LLMRequest{
		Model:        testModel(),
		SystemPrompt: "Be helpful.",
		Messages: []types.Message{
			&types.UserMessage{Role: "user", Content: []types.ContentBlock{
				&types.TextContent{Type: "text", Text: "msg1"},
			}},
		},
		Tools: []provider.ToolSchema{
			{Name: "read", Description: "Read a file", Parameters: map[string]any{"type": "object"}},
			{Name: "write", Description: "Write a file", Parameters: map[string]any{"type": "object"}},
		},
	}

	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	for range ch {
	}

	// Parse the captured request body.
	var body map[string]any
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	// System should be a structured array with cache_control.
	systemArr, ok := body["system"].([]any)
	if !ok {
		t.Fatalf("system should be []any, got %T: %v", body["system"], body["system"])
	}
	if len(systemArr) != 1 {
		t.Fatalf("expected 1 system block, got %d", len(systemArr))
	}
	sysBlock, ok := systemArr[0].(map[string]any)
	if !ok {
		t.Fatalf("system block should be map, got %T", systemArr[0])
	}
	if sysBlock["cache_control"] == nil {
		t.Error("system block should have cache_control")
	}
	if sysBlock["text"] != "Be helpful." {
		t.Errorf("system text: got %v", sysBlock["text"])
	}

	// Last tool should have cache_control.
	toolsArr, ok := body["tools"].([]any)
	if !ok || len(toolsArr) != 2 {
		t.Fatalf("expected 2 tools, got %v", body["tools"])
	}
	lastTool, ok := toolsArr[1].(map[string]any)
	if !ok {
		t.Fatalf("last tool should be map, got %T", toolsArr[1])
	}
	if lastTool["cache_control"] == nil {
		t.Error("last tool should have cache_control")
	}
	// First tool should NOT have cache_control.
	firstTool := toolsArr[0].(map[string]any)
	if firstTool["cache_control"] != nil {
		t.Error("first tool should NOT have cache_control")
	}
}
