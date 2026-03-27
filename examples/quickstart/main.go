// quickstart — the simplest pi-go program.
//
// Sends "1+1" to the LLM and prints the answer.
//
// Usage:
//
//	# Option A: ZenMux key (sk-ss-v1-...)
//	ZENMUX_API_KEY=sk-ss-v1-... go run ./examples/quickstart
//
//	# Option B: native Anthropic key (sk-ant-...)
//	ANTHROPIC_API_KEY=sk-ant-... go run ./examples/quickstart
//
//	# Option C: config file  ~/.pi-go/config.yml  (see config/example.yml)
//	go run ./examples/quickstart
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/config"
	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/provider/anthropic"
	"github.com/ai-gateway/pi-go/provider/zenmux"
	"github.com/ai-gateway/pi-go/types"
)

func main() {
	// Load config (falls back to env vars if file is absent).
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	prov, model := resolveProvider(cfg)

	a := agent.NewAgent(prov,
		agent.WithModel(model),
		agent.WithSystemPrompt("只回答数字，不要多余文字。"),
	)

	// Stream output to stdout.
	a.Subscribe(func(ev types.AgentEvent) {
		if ev.Type == types.EventMessageUpdate && ev.StreamEvent != nil &&
			ev.StreamEvent.Type == types.StreamEventTextDelta {
			fmt.Print(ev.StreamEvent.Delta)
		}
		if ev.Type == types.EventAgentEnd {
			fmt.Println()
		}
	})

	ctx := context.Background()
	if err := a.Prompt(ctx, "1+1"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	a.WaitForIdle(ctx)
}

// resolveProvider picks a provider + model from config/env in this order:
//  1. ZenMux  (ZENMUX_API_KEY or config providers.zenmux.api_key)
//  2. Anthropic  (ANTHROPIC_API_KEY / ANTHROPIC_AUTH_TOKEN or config providers.anthropic.api_key)
func resolveProvider(cfg *config.Config) (provider.LLMProvider, types.Model) {
	// ── ZenMux ──────────────────────────────────────────────────────────────
	if key := cfg.ProviderAPIKey("zenmux"); key != "" {
		baseURL := cfg.ProviderBaseURL("zenmux")
		if baseURL == "" {
			baseURL = zenmux.DefaultBaseURL
		}
		prov := zenmux.NewWithBaseURL(key, baseURL)
		modelID := cfg.DefaultModel
		if modelID == "" {
			modelID = "anthropic/claude-haiku-4-5"
		}
		return prov, types.Model{
			ID:        modelID,
			Provider:  "zenmux",
			BaseURL:   baseURL,
			MaxTokens: 256,
		}
	}

	// ── Anthropic (native or via proxy) ─────────────────────────────────────
	key := cfg.ProviderAPIKey("anthropic")
	if key == "" {
		// Also accept ANTHROPIC_AUTH_TOKEN (ZenMux session token for Anthropic proxy).
		key = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "no API key found — set ZENMUX_API_KEY or ANTHROPIC_API_KEY")
		os.Exit(1)
	}

	baseURL := cfg.ProviderBaseURL("anthropic")
	if baseURL == "" {
		// ANTHROPIC_AUTH_TOKEN (sk-ss-v1-) → ZenMux Anthropic proxy
		if isZenMuxToken(key) {
			baseURL = "https://zenmux.ai/api/anthropic"
		} else {
			baseURL = "https://api.anthropic.com"
		}
	}

	prov := anthropic.NewWithBaseURL(key, baseURL)
	return prov, types.Model{
		ID:        "claude-haiku-4-5-20251001",
		Provider:  "anthropic",
		BaseURL:   baseURL,
		MaxTokens: 256,
	}
}

// isZenMuxToken reports whether key is a ZenMux session/subscription token
// (starts with "sk-ss-") rather than a native Anthropic key ("sk-ant-").
func isZenMuxToken(key string) bool {
	return len(key) > 6 && key[:6] == "sk-ss-"
}
