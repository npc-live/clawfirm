// quickstart — the simplest clawfirm program.
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
//	# Option C: config file  ~/.clawfirm/config.yml  (see config/example.yml)
//	go run ./examples/quickstart
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ai-gateway/clawfirm/agent"
	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/provider/anthropic"
	"github.com/ai-gateway/clawfirm/types"
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
	// Try ZENMUX_API_KEY first (via ZenMux Anthropic proxy), then ANTHROPIC_API_KEY.
	key := cfg.ProviderAPIKey("zenmux")
	baseURL := cfg.ProviderBaseURL("zenmux")
	if key == "" {
		key = cfg.ProviderAPIKey("anthropic")
		baseURL = cfg.ProviderBaseURL("anthropic")
	}
	if key == "" {
		key = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "no API key found — set ZENMUX_API_KEY or ANTHROPIC_API_KEY")
		os.Exit(1)
	}

	if baseURL == "" {
		if len(key) > 6 && key[:6] == "sk-ss-" {
			baseURL = "https://zenmux.ai/api/anthropic"
		} else {
			baseURL = "https://api.anthropic.com"
		}
	}

	prov := anthropic.NewWithBaseURL(key, baseURL)
	modelID := cfg.DefaultModel
	if modelID == "" {
		modelID = "claude-haiku-4-5-20251001"
	}
	return prov, types.Model{
		ID:        modelID,
		Provider:  "anthropic",
		BaseURL:   baseURL,
		MaxTokens: 256,
	}
}
