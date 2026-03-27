package openai

import "github.com/ai-gateway/pi-go/types"

// BuiltinModels returns the hard-coded list of OpenAI models.
func BuiltinModels() []types.Model {
	return []types.Model{
		{
			ID:            "gpt-5.4",
			Name:          "GPT-5.4",
			Provider:      "openai",
			BaseURL:       "https://api.openai.com",
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 2.5, Output: 10.0},
			ContextWindow: 128000,
			MaxTokens:     16384,
		},
		{
			ID:            "gpt-4o",
			Name:          "GPT-4o",
			Provider:      "openai",
			BaseURL:       "https://api.openai.com",
			Reasoning:     false,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 2.5, Output: 10.0},
			ContextWindow: 128000,
			MaxTokens:     16384,
		},
		{
			ID:            "o3",
			Name:          "O3",
			Provider:      "openai",
			BaseURL:       "https://api.openai.com",
			Reasoning:     true,
			InputTypes:    []string{"text", "image"},
			Cost:          types.ModelCost{Input: 10.0, Output: 40.0},
			ContextWindow: 200000,
			MaxTokens:     100000,
		},
	}
}
