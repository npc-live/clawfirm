package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
)

func testModel(baseURL string) types.Model {
	return types.Model{
		ID:       "llama3",
		Provider: "ollama",
		BaseURL:  baseURL,
	}
}

func makeRequest(model types.Model) provider.LLMRequest {
	return provider.LLMRequest{
		Model: model,
		Messages: []types.Message{
			&types.UserMessage{
				Role:    "user",
				Content: []types.ContentBlock{&types.TextContent{Type: "text", Text: "hello"}},
			},
		},
	}
}

func buildTextNDJSON(text string) string {
	chunks := ""
	for i, ch := range text {
		last := i == len([]rune(text))-1
		if last {
			chunks += fmt.Sprintf("{\"model\":\"llama3\",\"message\":{\"role\":\"assistant\",\"content\":%q},\"done\":true}\n", string(ch))
		} else {
			chunks += fmt.Sprintf("{\"model\":\"llama3\",\"message\":{\"role\":\"assistant\",\"content\":%q},\"done\":false}\n", string(ch))
		}
	}
	return chunks
}

func TestOllamaTextResponse(t *testing.T) {
	wantText := "Hi!"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildTextNDJSON(wantText))
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	ch, err := p.Stream(context.Background(), makeRequest(testModel(srv.URL)))
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
	if fullText != wantText {
		t.Errorf("text: got %q want %q", fullText, wantText)
	}
	if doneMsg == nil {
		t.Fatal("expected done event")
	}
	if doneMsg.StopReason != types.StopReasonStop {
		t.Errorf("StopReason: got %q want stop", doneMsg.StopReason)
	}
}

func TestOllamaAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"model not found"}`)
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	_, err := p.Stream(context.Background(), makeRequest(testModel(srv.URL)))
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestOllamaCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"model":"llama3","message":{"role":"assistant","content":"hi"},"done":false}`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	p := NewWithBaseURL(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.Stream(ctx, makeRequest(testModel(srv.URL)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	<-ch
	cancel()
	for range ch {
	}
}

func TestOllamaDiscoverModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"models":[{"name":"llama3"},{"name":"mistral"}]}`)
	}))
	defer srv.Close()

	models, err := DiscoverModels(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("DiscoverModels error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "llama3" {
		t.Errorf("models[0].ID: got %q want llama3", models[0].ID)
	}
}
