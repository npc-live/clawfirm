package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// ollamaTagsResponse is the response from the /api/tags endpoint.
type ollamaTagsResponse struct {
	Models []struct {
		Name       string    `json:"name"`
		ModifiedAt time.Time `json:"modified_at"`
		Size       int64     `json:"size"`
	} `json:"models"`
}

// DiscoverModels fetches the list of available models from the Ollama server.
func DiscoverModels(ctx context.Context, baseURL string) ([]types.Model, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := baseURL + "/api/tags"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ollama: discover request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: discover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ollama: discover HTTP %d", resp.StatusCode)
	}

	var tagsResp ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("ollama: decode tags: %w", err)
	}

	models := make([]types.Model, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		models = append(models, types.Model{
			ID:         m.Name,
			Name:       m.Name,
			Provider:   "ollama",
			BaseURL:    baseURL,
			InputTypes: []string{"text"},
		})
	}
	return models, nil
}
