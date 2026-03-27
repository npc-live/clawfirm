package zenmux_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/provider/zenmux"
	"github.com/ai-gateway/pi-go/types"
)

// sseResponse builds a minimal OpenAI-compatible SSE response.
func sseResponse(text string) string {
	type delta struct {
		Content *string `json:"content,omitempty"`
	}
	type choice struct {
		Index        int     `json:"index"`
		Delta        delta   `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	}
	type chunk struct {
		ID      string   `json:"id"`
		Object  string   `json:"object"`
		Choices []choice `json:"choices"`
	}

	stop := "stop"
	content := text

	c1, _ := json.Marshal(chunk{
		ID:     "test-id",
		Object: "chat.completion.chunk",
		Choices: []choice{
			{Index: 0, Delta: delta{Content: &content}},
		},
	})
	c2, _ := json.Marshal(chunk{
		ID:     "test-id",
		Object: "chat.completion.chunk",
		Choices: []choice{
			{Index: 0, Delta: delta{}, FinishReason: &stop},
		},
	})
	return fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: [DONE]\n\n", c1, c2)
}

func TestZenmuxStreamText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseResponse("hello from zenmux"))
	}))
	defer srv.Close()

	p := zenmux.NewWithBaseURL("test-key", srv.URL)

	req := provider.LLMRequest{
		Model: types.Model{
			ID:       "openai/gpt-4o",
			Provider: "zenmux",
			BaseURL:  srv.URL,
		},
		Messages: []types.Message{
			&types.UserMessage{
				Role:    "user",
				Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "hi"}},
			},
		},
	}

	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	var gotText string
	var gotDone bool
	for ev := range ch {
		switch ev.Type {
		case types.StreamEventTextDelta:
			gotText += ev.Delta
		case types.StreamEventDone:
			gotDone = true
			if ev.Message == nil {
				t.Error("done event missing message")
			} else if ev.Message.Provider != "zenmux" {
				t.Errorf("provider: got %q want zenmux", ev.Message.Provider)
			}
		}
	}

	if gotText != "hello from zenmux" {
		t.Errorf("text: got %q want %q", gotText, "hello from zenmux")
	}
	if !gotDone {
		t.Error("missing StreamEventDone")
	}
}

func TestZenmuxID(t *testing.T) {
	p := zenmux.New("key")
	if p.ID() != "zenmux" {
		t.Errorf("ID: got %q want zenmux", p.ID())
	}
}

func TestZenmuxModels(t *testing.T) {
	models := zenmux.BuiltinModels()
	if len(models) == 0 {
		t.Fatal("expected at least one model")
	}
	for _, m := range models {
		if m.Provider != "zenmux" {
			t.Errorf("model %q: provider=%q want zenmux", m.ID, m.Provider)
		}
		if m.BaseURL != zenmux.DefaultBaseURL {
			t.Errorf("model %q: baseURL=%q want %q", m.ID, m.BaseURL, zenmux.DefaultBaseURL)
		}
	}
}

func TestIsQuotaRefreshError(t *testing.T) {
	cases := []struct {
		status  int
		message string
		want    bool
	}{
		// ZenMux rolling-window quota message (openclaw#43917)
		{
			status: 402,
			message: "You have reached your subscription quota limit. " +
				"Please wait for automatic quota refresh in the rolling time window, " +
				"upgrade to a higher plan, or use a Pay-As-You-Go API Key for unlimited access. " +
				"Learn more: https://zenmux.ai/docs/guide/subscription.html",
			want: true,
		},
		// Other 402 billing errors should NOT be quota-refresh
		{status: 402, message: "Payment required", want: false},
		{status: 402, message: "Your credit balance is too low", want: false},
		// Non-402 status codes are never quota-refresh
		{status: 429, message: "subscription quota limit", want: false},
		{status: 200, message: "subscription quota limit", want: false},
	}

	for _, c := range cases {
		got := zenmux.IsQuotaRefreshError(c.status, c.message)
		if got != c.want {
			t.Errorf("IsQuotaRefreshError(%d, %q) = %v, want %v", c.status, c.message, got, c.want)
		}
	}
}
