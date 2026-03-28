package gemini

import "github.com/ai-gateway/pi-go/types"

// BuiltinModels returns the hard-coded list of Gemini models.
func BuiltinModels() []types.Model {
	return []types.Model{
		{
			ID:            "gemini-2.5-pro",
			Name:          "Gemini 2.5 Pro",
			Provider:      "gemini",
			BaseURL:       "https://generativelanguage.googleapis.com",
			Reasoning:     true,
			InputTypes:    []string{"text", "image", "audio"},
			Cost:          types.ModelCost{Input: 1.25, Output: 5.0},
			ContextWindow: 2000000,
			MaxTokens:     8192,
		},
		{
			ID:            "gemini-2.0-flash",
			Name:          "Gemini 2.0 Flash",
			Provider:      "gemini",
			BaseURL:       "https://generativelanguage.googleapis.com",
			Reasoning:     false,
			InputTypes:    []string{"text", "image", "audio"},
			Cost:          types.ModelCost{Input: 0.1, Output: 0.4},
			ContextWindow: 1000000,
			MaxTokens:     8192,
		},
	}
}
