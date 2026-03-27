package anthropic

import "github.com/ai-gateway/pi-go/types"

// BuiltinModels returns the hard-coded list of Anthropic models.
func BuiltinModels() []types.Model {
	return []types.Model{
		{
			ID:            "claude-opus-4-6",
			Name:          "Claude Opus 4.6",
			Provider:      "anthropic",
			BaseURL:       "https://api.anthropic.com",
			Reasoning:     true,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 15.0, Output: 75.0, CacheRead: 1.5, CacheWrite: 18.75},
			ContextWindow: 200000,
			MaxTokens:     32000,
		},
		{
			ID:            "claude-sonnet-4-6",
			Name:          "Claude Sonnet 4.6",
			Provider:      "anthropic",
			BaseURL:       "https://api.anthropic.com",
			Reasoning:     true,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
			ContextWindow: 200000,
			MaxTokens:     64000,
		},
		{
			ID:            "claude-haiku-4-5",
			Name:          "Claude Haiku 4.5",
			Provider:      "anthropic",
			BaseURL:       "https://api.anthropic.com",
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 0.8, Output: 4.0, CacheRead: 0.08, CacheWrite: 1.0},
			ContextWindow: 200000,
			MaxTokens:     8192,
		},
	}
}
