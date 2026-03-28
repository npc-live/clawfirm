//go:build live

package anthropic_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/provider/anthropic"
	"github.com/ai-gateway/pi-go/types"
)

// TestLiveDiagDirect calls the Anthropic provider directly (bypassing the agent)
// to capture the raw HTTP error.  Run with:
//
//	ANTHROPIC_AUTH_TOKEN=sk-ss-v1-... go test ./provider/anthropic/ -run TestLiveDiagDirect -v
func TestLiveDiagDirect(t *testing.T) {
	var apiKey, baseURL string

	if tok := os.Getenv("ANTHROPIC_AUTH_TOKEN"); tok != "" {
		apiKey = tok
		// ZenMux session key — use ZenMux Anthropic proxy endpoint.
		baseURL = "https://zenmux.ai/api/anthropic"
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		apiKey = key
		baseURL = "https://api.anthropic.com"
	} else {
		t.Skip("no API key set")
	}

	t.Logf("using baseURL: %s, key prefix: %s...", baseURL, safePrefix(apiKey, 12))

	prov := anthropic.NewWithBaseURL(apiKey, baseURL)

	model := types.Model{
		ID:        "claude-haiku-4-5-20251001",
		Name:      "Claude Haiku 4.5",
		Provider:  "anthropic",
		BaseURL:   baseURL,
		MaxTokens: 256,
	}

	req := provider.LLMRequest{
		Model: model,
		Messages: []types.Message{
			&types.UserMessage{
				Role:    "user",
				Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "1+1=?"}},
			},
		},
		SystemPrompt: "Reply in one word.",
	}

	ch, err := prov.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}

	for ev := range ch {
		t.Logf("event type=%s delta=%q", ev.Type, ev.Delta)
		if ev.Error != nil {
			t.Errorf("stream error: provider=%s msg=%s", ev.Error.Provider, ev.Error.ErrorMessage)
		}
		if ev.Message != nil {
			t.Logf("done: stop_reason=%s tokens=%+v", ev.Message.StopReason, ev.Message.Usage)
		}
	}
}

// TestLiveMiniMax tests MiniMax via its Anthropic-compatible API.
// Run with:
//
//	MINIMAX_API_KEY=sk-cp--... go test ./provider/anthropic/ -run TestLiveMiniMax -v -tags live
func TestLiveMiniMax(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY not set")
	}
	const baseURL = "https://api.minimax.io/anthropic"
	t.Logf("using MiniMax baseURL: %s, key prefix: %s...", baseURL, safePrefix(apiKey, 12))

	prov := anthropic.NewWithBaseURL(apiKey, baseURL)

	model := types.Model{
		ID:        "MiniMax-M2.7",
		Name:      "MiniMax M2.7",
		Provider:  "minimax",
		BaseURL:   baseURL,
		MaxTokens: 256,
	}

	req := provider.LLMRequest{
		Model: model,
		Messages: []types.Message{
			&types.UserMessage{
				Role:    "user",
				Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "1+1等于几？用中文一句话回答"}},
			},
		},
		SystemPrompt: "你是一个简洁的助手。",
	}

	ch, err := prov.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}

	var text string
	for ev := range ch {
		if ev.Type == types.StreamEventTextDelta {
			text += ev.Delta
			t.Logf("delta: %q", ev.Delta)
		}
		if ev.Error != nil {
			t.Errorf("stream error: %s", ev.Error.ErrorMessage)
		}
		if ev.Message != nil {
			t.Logf("done: stop_reason=%s content_len=%d tokens=%+v",
				ev.Message.StopReason, len(ev.Message.Content), ev.Message.Usage)
		}
	}
	t.Logf("full response: %s", text)
	if text == "" {
		t.Error("expected non-empty text response")
	}
}

func safePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return strings.Repeat("*", n)
}
