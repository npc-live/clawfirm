package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// WebSearch performs web searches via the Brave Search API.
// If APIKey is empty, the tool returns an error prompting configuration.
type WebSearch struct {
	// APIKey is the Brave Search API key (or set via BRAVE_SEARCH_API_KEY env).
	APIKey string
	// BaseURL overrides the Brave API base URL (for testing).
	BaseURL string
}

func (w *WebSearch) Name() string  { return "web_search" }
func (w *WebSearch) Label() string { return "Web Search" }

func (w *WebSearch) Description() string {
	return "Search the web for current information. Returns top results with titles, URLs, and snippets. " +
		"Use when you need up-to-date information, documentation, error solutions, or facts beyond your training data."
}

func (w *WebSearch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query string.",
			},
			"num_results": map[string]any{
				"type":        "integer",
				"description": "Number of results to return (default 5, max 10).",
			},
		},
		"required": []string{"query"},
	}
}

func (w *WebSearch) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	query, _ := params["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return tool.ToolResult{}, fmt.Errorf("web_search: query is required")
	}

	numResults := 5
	if n, ok := params["num_results"].(float64); ok && n >= 1 && n <= 10 {
		numResults = int(n)
	}

	if w.APIKey == "" {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: "web_search is not configured. Set BRAVE_SEARCH_API_KEY to enable web search."},
			},
		}, nil
	}

	baseURL := w.BaseURL
	if baseURL == "" {
		baseURL = "https://api.search.brave.com"
	}
	results, err := braveSearch(ctx, baseURL, w.APIKey, query, numResults)
	if err != nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: fmt.Sprintf("Search failed: %v", err)},
			},
		}, nil
	}

	if len(results) == 0 {
		return tool.ToolResult{
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText,
					Text: "No results found for: " + query},
			},
		}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Search results for %q:\n\n", query)
	for i, r := range results {
		fmt.Fprintf(&sb, "%d. **%s**\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet)
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: sb.String()},
		},
	}, nil
}

// searchResult holds a single search result.
type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

// braveSearch queries the Brave Search API and returns parsed results.
func braveSearch(ctx context.Context, baseURL, apiKey, query string, count int) ([]searchResult, error) {
	u := fmt.Sprintf("%s/res/v1/web/search?q=%s&count=%d",
		baseURL, url.QueryEscape(query), count)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("Brave API HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	results := make([]searchResult, 0, len(apiResp.Web.Results))
	for _, r := range apiResp.Web.Results {
		results = append(results, searchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
		})
	}
	return results, nil
}
