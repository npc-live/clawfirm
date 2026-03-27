package zenmux

import "github.com/ai-gateway/pi-go/types"

// BuiltinModels returns the default ZenMux model list.
// ZenMux aggregates models from multiple providers; slugs use the format
// "provider/model-id". See https://zenmux.ai/models for the full catalog.
func BuiltinModels() []types.Model {
	return []types.Model{
		// ── OpenAI ──────────────────────────────────────────────────────────
		{
			ID:            "openai/gpt-5",
			Name:          "GPT-5",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 10.0, Output: 30.0},
			ContextWindow: 128000,
			MaxTokens:     16384,
		},
		{
			ID:            "openai/gpt-4o",
			Name:          "GPT-4o",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 2.5, Output: 10.0},
			ContextWindow: 128000,
			MaxTokens:     16384,
		},
		{
			ID:            "openai/gpt-4o-mini",
			Name:          "GPT-4o Mini",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 0.15, Output: 0.6},
			ContextWindow: 128000,
			MaxTokens:     16384,
		},
		// ── Anthropic ───────────────────────────────────────────────────────
		{
			ID:            "anthropic/claude-sonnet-4-5",
			Name:          "Claude Sonnet 4.5",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     true,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 3.0, Output: 15.0},
			ContextWindow: 200000,
			MaxTokens:     64000,
		},
		{
			ID:            "anthropic/claude-haiku-4-5",
			Name:          "Claude Haiku 4.5",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 0.8, Output: 4.0},
			ContextWindow: 200000,
			MaxTokens:     8192,
		},
		// ── Google ──────────────────────────────────────────────────────────
		{
			ID:            "google/gemini-2.5-pro",
			Name:          "Gemini 2.5 Pro",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     true,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 1.25, Output: 10.0},
			ContextWindow: 1000000,
			MaxTokens:     65536,
		},
		{
			ID:            "google/gemini-2.5-flash",
			Name:          "Gemini 2.5 Flash",
			Provider:      "zenmux",
			BaseURL:       DefaultBaseURL,
			Reasoning:     true,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 0.15, Output: 0.6},
			ContextWindow: 1000000,
			MaxTokens:     65536,
		},
	}
}
