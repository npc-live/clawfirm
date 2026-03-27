package gemini

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
		ID:        "gemini-2.0-flash",
		Provider:  "gemini",
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
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":%q}],\"role\":\"model\"},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":10,\"candidatesTokenCount\":5}}\n\n",
		text,
	)
}

func buildToolCallSSE(toolName string, argsJSON string) string {
	return fmt.Sprintf(
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":%q,\"args\":%s}}],\"role\":\"model\"},\"finishReason\":\"STOP\"}]}\n\n",
		toolName, argsJSON,
	)
}

func TestGeminiTextResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildTextSSE("Hello from Gemini"))
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
	if fullText != "Hello from Gemini" {
		t.Errorf("text: got %q want %q", fullText, "Hello from Gemini")
	}
	if doneMsg == nil {
		t.Fatal("expected done event")
	}
	if doneMsg.StopReason != types.StopReasonStop {
		t.Errorf("StopReason: got %q want stop", doneMsg.StopReason)
	}
	if doneMsg.Usage.Input != 10 {
		t.Errorf("Usage.Input: got %d want 10", doneMsg.Usage.Input)
	}
}

func TestGeminiToolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildToolCallSSE("search", `{"query":"go testing"}`))
	}))
	defer srv.Close()

	p := NewWithBaseURL("test-key", srv.URL)
	ch, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	var toolCallEnd *types.AssistantMessageEvent
	for ev := range ch {
		evCopy := ev
		if ev.Type == types.StreamEventToolCallEnd {
			toolCallEnd = &evCopy
		}
	}

	if toolCallEnd == nil {
		t.Fatal("expected toolcall_end event")
	}
	if toolCallEnd.ToolCall == nil {
		t.Fatal("ToolCall is nil")
	}
	if toolCallEnd.ToolCall.Name != "search" {
		t.Errorf("ToolCall.Name: got %q want search", toolCallEnd.ToolCall.Name)
	}
	q, _ := toolCallEnd.ToolCall.Arguments["query"].(string)
	if q != "go testing" {
		t.Errorf("ToolCall.Arguments.query: got %q want 'go testing'", q)
	}
}

func TestGeminiAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":{"message":"API key invalid","status":"PERMISSION_DENIED"}}`)
	}))
	defer srv.Close()

	p := NewWithBaseURL("bad-key", srv.URL)
	_, err := p.Stream(context.Background(), makeRequest(testModel()))
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGeminiCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"candidates\":[]}\n\n")
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

	<-ch
	cancel()
	for range ch {
	}
}

func TestGeminiModels(t *testing.T) {
	p := New("key")
	models := p.Models()
	if len(models) == 0 {
		t.Error("expected non-empty models list")
	}
	for _, m := range models {
		if m.Provider != "gemini" {
			t.Errorf("model %q Provider = %q, want gemini", m.ID, m.Provider)
		}
	}
}
