package openai

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
		ID:        "gpt-5.4",
		Provider:  "openai",
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

func buildTextSSE(text string) string {
	return fmt.Sprintf(
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":%q},\"finish_reason\":null}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"+
			"data: [DONE]\n\n",
		text,
	)
}

func buildToolCallSSE(toolID, toolName, argsJSON string) string {
	return fmt.Sprintf(
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":%q,\"type\":\"function\",\"function\":{\"name\":%q,\"arguments\":\"\"}}]},\"finish_reason\":null}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":%q}}]},\"finish_reason\":null}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n"+
			"data: [DONE]\n\n",
		toolID, toolName, argsJSON,
	)
}

func TestOpenAITextResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildTextSSE("Hello from OpenAI"))
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
	if fullText != "Hello from OpenAI" {
		t.Errorf("text: got %q want %q", fullText, "Hello from OpenAI")
	}
	if doneMsg == nil {
		t.Fatal("expected done event")
	}
	if doneMsg.StopReason != types.StopReasonStop {
		t.Errorf("StopReason: got %q want stop", doneMsg.StopReason)
	}
}

func TestOpenAIToolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildToolCallSSE("call_xyz", "search", `{"query":"golang"}`))
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
	if doneMsg == nil {
		t.Fatal("expected done message")
	}
	if doneMsg.StopReason != types.StopReasonToolUse {
		t.Errorf("StopReason: got %q want toolUse", doneMsg.StopReason)
	}
}

func TestOpenAIAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`)
	}))
	defer srv.Close()

	p := NewWithBaseURL("bad-key", srv.URL)
	_, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestOpenAICtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.Stream(ctx, makeRequest(testModel()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for at least one event
	<-ch
	cancel()
	for range ch {
	}
}

func TestOpenAIModels(t *testing.T) {
	p := New("key")
	models := p.Models()
	if len(models) == 0 {
		t.Error("expected non-empty models list")
	}
	for _, m := range models {
		if m.Provider != "openai" {
			t.Errorf("model %q Provider = %q, want openai", m.ID, m.Provider)
		}
	}
}
