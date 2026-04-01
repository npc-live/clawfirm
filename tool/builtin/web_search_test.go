package builtin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

func TestWebSearch_MissingQuery(t *testing.T) {
	ws := &WebSearch{APIKey: "key"}
	_, err := ws.Execute(context.Background(), "1", map[string]any{}, nopUpdate)
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Errorf("expected query-required error, got %v", err)
	}
}

func TestWebSearch_NoAPIKey(t *testing.T) {
	ws := &WebSearch{}
	res, err := ws.Execute(context.Background(), "1", map[string]any{"query": "test"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "not configured") {
		t.Errorf("expected not-configured message, got %q", text)
	}
}

func TestWebSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-key" {
			t.Errorf("expected API key header, got %q", r.Header.Get("X-Subscription-Token"))
		}
		if !strings.Contains(r.URL.Path, "/res/v1/web/search") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"web": {
				"results": [
					{"title": "Go Testing", "url": "https://go.dev/doc/testing", "description": "How to test in Go."},
					{"title": "Go Blog", "url": "https://go.dev/blog", "description": "The Go blog."}
				]
			}
		}`)
	}))
	defer srv.Close()

	ws := &WebSearch{APIKey: "test-key", BaseURL: srv.URL}
	res, err := ws.Execute(context.Background(), "1", map[string]any{"query": "go testing"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "Go Testing") {
		t.Errorf("expected result title, got %q", text)
	}
	if !strings.Contains(text, "go.dev/doc/testing") {
		t.Errorf("expected result URL, got %q", text)
	}
}

func TestWebSearch_NoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"web":{"results":[]}}`)
	}))
	defer srv.Close()

	ws := &WebSearch{APIKey: "key", BaseURL: srv.URL}
	res, err := ws.Execute(context.Background(), "1", map[string]any{"query": "nonexistent"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "No results found") {
		t.Errorf("expected no-results message, got %q", text)
	}
}

func TestWebSearch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"rate limited"}`)
	}))
	defer srv.Close()

	ws := &WebSearch{APIKey: "key", BaseURL: srv.URL}
	res, err := ws.Execute(context.Background(), "1", map[string]any{"query": "test"}, nopUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "Search failed") {
		t.Errorf("expected failure message, got %q", text)
	}
}

func TestWebSearch_NumResults(t *testing.T) {
	var capturedCount string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCount = r.URL.Query().Get("count")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"web":{"results":[]}}`)
	}))
	defer srv.Close()

	ws := &WebSearch{APIKey: "key", BaseURL: srv.URL}
	_, _ = ws.Execute(context.Background(), "1", map[string]any{
		"query":       "test",
		"num_results": float64(3),
	}, nopUpdate)
	if capturedCount != "3" {
		t.Errorf("count param: got %q want 3", capturedCount)
	}
}

func TestWebSearch_Schema(t *testing.T) {
	ws := &WebSearch{}
	s := ws.Schema()
	if s["type"] != "object" {
		t.Errorf("schema type: got %v want object", s["type"])
	}
	props := s["properties"].(map[string]any)
	if _, ok := props["query"]; !ok {
		t.Error("schema missing query property")
	}
	req := s["required"].([]string)
	if len(req) != 1 || req[0] != "query" {
		t.Errorf("required: got %v want [query]", req)
	}
}

func TestWebSearch_NameAndLabel(t *testing.T) {
	ws := &WebSearch{}
	if ws.Name() != "web_search" {
		t.Errorf("Name: got %q", ws.Name())
	}
	if ws.Label() == "" {
		t.Error("Label should not be empty")
	}
}

func nopUpdate(tool.ToolUpdate) {}

func textFromResult(res tool.ToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*types.TextContent); ok {
		return tc.Text
	}
	return ""
}
