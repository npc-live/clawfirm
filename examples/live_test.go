//go:build live

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/provider/anthropic"
	"github.com/ai-gateway/pi-go/types"
)

// TestLiveAnthropic tests the agent against the Anthropic API.
//
// ANTHROPIC_AUTH_TOKEN (sk-ss-v1-... ZenMux key): routed through ZenMux's
// Anthropic-compatible proxy at https://zenmux.ai/api/anthropic.
//
// ANTHROPIC_API_KEY (sk-ant-... native key): routed directly to api.anthropic.com.
func TestLiveAnthropic(t *testing.T) {
	var apiKey, baseURL string

	if tok := os.Getenv("ANTHROPIC_AUTH_TOKEN"); tok != "" {
		// ZenMux session key — use ZenMux Anthropic proxy endpoint.
		apiKey = tok
		baseURL = "https://zenmux.ai/api/anthropic"
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		apiKey = key
		baseURL = "https://api.anthropic.com"
	} else {
		t.Skip("no API key set (ANTHROPIC_AUTH_TOKEN or ANTHROPIC_API_KEY)")
	}

	prov := anthropic.NewWithBaseURL(apiKey, baseURL)

	// claude-haiku-4-5 — fast and cheap for a live smoke test.
	model := types.Model{
		ID:        "claude-haiku-4-5-20251001",
		Name:      "Claude Haiku 4.5",
		Provider:  "anthropic",
		BaseURL:   baseURL,
		MaxTokens: 256,
	}

	a := agent.NewAgent(prov,
		agent.WithModel(model),
		agent.WithSystemPrompt("用中文回复，保持简短。"),
	)

	var output string
	a.Subscribe(func(ev types.AgentEvent) {
		t.Logf("event: %s", ev.Type)
		if ev.Type == types.EventMessageUpdate && ev.StreamEvent != nil {
			t.Logf("  stream event: %s delta=%q", ev.StreamEvent.Type, ev.StreamEvent.Delta)
			if ev.StreamEvent.Type == types.StreamEventTextDelta {
				fmt.Print(ev.StreamEvent.Delta)
				output += ev.StreamEvent.Delta
			}
		}
		if ev.Type == types.EventAgentEnd {
			fmt.Println()
			if len(ev.Messages) > 0 {
				last := ev.Messages[len(ev.Messages)-1]
				if am, ok := last.(*types.AssistantMessage); ok {
					t.Logf("  stop_reason=%s error=%q", am.StopReason, am.ErrorMessage)
				}
			}
		}
	})

	ctx := context.Background()
	if err := a.Prompt(ctx, "1+1等于几？只回答数字"); err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if err := a.WaitForIdle(ctx); err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}

	if output == "" {
		t.Error("no output received")
	}
	t.Logf("response: %q", output)
}
