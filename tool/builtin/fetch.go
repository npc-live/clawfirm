package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

const fetchMaxBytes = 512 * 1024 // 512 KB

// Fetch makes an HTTP request and returns the response body.
type Fetch struct{}

func (f *Fetch) Name() string  { return "fetch" }
func (f *Fetch) Label() string { return "Fetch" }
func (f *Fetch) Description() string {
	return "Make an HTTP request (GET/POST/PUT/DELETE/etc.) and return the response body."
}
func (f *Fetch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to request.",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method (default: GET).",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "HTTP headers as key-value pairs.",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Request body (for POST/PUT).",
			},
			"json": map[string]any{
				"type":        "object",
				"description": "JSON body (sets Content-Type: application/json automatically).",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default 30).",
			},
		},
		"required": []string{"url"},
	}
}

func (f *Fetch) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	url, _ := params["url"].(string)
	if url == "" {
		return tool.ToolResult{}, fmt.Errorf("fetch: url is required")
	}

	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)

	timeoutSecs := 30.0
	if t, ok := params["timeout"].(float64); ok && t > 0 {
		timeoutSecs = t
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs*float64(time.Second)))
	defer cancel()

	// Build request body.
	var bodyReader io.Reader
	contentType := ""

	if jsonBody, ok := params["json"].(map[string]any); ok {
		data, err := json.Marshal(jsonBody)
		if err != nil {
			return tool.ToolResult{}, fmt.Errorf("fetch: json marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
		contentType = "application/json"
	} else if bodyStr, ok := params["body"].(string); ok && bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("fetch: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if headers, ok := params["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBytes))
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("fetch: read body: %w", err)
	}

	truncated := ""
	if len(body) >= fetchMaxBytes {
		truncated = "\n[Response truncated to 512 KB]"
	}

	text := fmt.Sprintf("HTTP %d %s\n\n%s%s", resp.StatusCode, resp.Status, string(body), truncated)

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}, nil
}
