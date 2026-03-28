package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// EmbeddingProvider generates dense vector embeddings for text.
type EmbeddingProvider interface {
	// Name returns the provider identifier, e.g. "openai".
	Name() string
	// Model returns the model identifier used for embedding.
	Model() string
	// Dims returns the vector dimension produced by this provider.
	Dims() int
	// Embed encodes a batch of texts and returns one float32 vector per text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// NewAutoProvider returns the first available EmbeddingProvider by probing env
// vars and a local Ollama instance, in priority order:
//
//  1. OpenRouter (OPENROUTER_API_KEY)
//  2. OpenAI     (OPENAI_API_KEY)
//  3. Gemini     (GEMINI_API_KEY)
//  4. Voyage     (VOYAGE_API_KEY)
//  5. Mistral    (MISTRAL_API_KEY)
//  6. Ollama     (local, no key)
//
// Returns an error when none is reachable.
func NewAutoProvider() (EmbeddingProvider, error) {
	if k := os.Getenv("OPENROUTER_API_KEY"); k != "" {
		return &OpenRouterEmbedder{APIKey: k}, nil
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		return &OpenAIEmbedder{APIKey: k}, nil
	}
	if k := os.Getenv("GEMINI_API_KEY"); k != "" {
		return &GeminiEmbedder{APIKey: k}, nil
	}
	if k := os.Getenv("VOYAGE_API_KEY"); k != "" {
		return &VoyageEmbedder{APIKey: k}, nil
	}
	if k := os.Getenv("MISTRAL_API_KEY"); k != "" {
		return &MistralEmbedder{APIKey: k}, nil
	}
	// Probe Ollama with a short timeout.
	ol := &OllamaEmbedder{BaseURL: "http://localhost:11434"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := ol.Embed(ctx, []string{"ping"}); err == nil {
		return ol, nil
	}
	return nil, fmt.Errorf(
		"memory: no embedding provider available — set OPENAI_API_KEY, GEMINI_API_KEY, " +
			"VOYAGE_API_KEY, MISTRAL_API_KEY, or run Ollama locally",
	)
}

// ─── OpenRouter ───────────────────────────────────────────────────────────────

// OpenRouterEmbedder proxies to OpenRouter's embeddings endpoint, which is
// API-compatible with OpenAI. Uses openai/text-embedding-3-small (1536 dims).
type OpenRouterEmbedder struct {
	APIKey  string
	BaseURL string // defaults to https://openrouter.ai
	client  *http.Client
}

func (e *OpenRouterEmbedder) Name() string  { return "openrouter" }
func (e *OpenRouterEmbedder) Model() string { return "openai/text-embedding-3-small" }
func (e *OpenRouterEmbedder) Dims() int     { return 1536 }

func (e *OpenRouterEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base := e.BaseURL
	if base == "" {
		base = "https://openrouter.ai"
	}
	type req struct {
		Input          []string `json:"input"`
		Model          string   `json:"model"`
		EncodingFormat string   `json:"encoding_format"`
	}
	type embObj struct {
		Embedding []float32 `json:"embedding"`
	}
	type resp struct {
		Data  []embObj `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	body, _ := json.Marshal(req{Input: texts, Model: e.Model(), EncodingFormat: "float"})
	r, err := httpPost(ctx, e.httpClient(), base+"/api/v1/embeddings",
		map[string]string{"Authorization": "Bearer " + e.APIKey}, body)
	if err != nil {
		return nil, fmt.Errorf("openrouter embed: %w", err)
	}
	var out resp
	if err := json.Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("openrouter embed decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("openrouter embed: %s", out.Error.Message)
	}
	result := make([][]float32, len(out.Data))
	for i, d := range out.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

func (e *OpenRouterEmbedder) httpClient() *http.Client {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}
	return e.client
}

// ─── OpenAI ──────────────────────────────────────────────────────────────────

// OpenAIEmbedder uses text-embedding-3-small (1536 dims).
type OpenAIEmbedder struct {
	APIKey  string
	BaseURL string // defaults to https://api.openai.com
	client  *http.Client
}

func (e *OpenAIEmbedder) Name() string { return "openai" }
func (e *OpenAIEmbedder) Model() string { return "text-embedding-3-small" }
func (e *OpenAIEmbedder) Dims() int    { return 1536 }

func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base := e.BaseURL
	if base == "" {
		base = "https://api.openai.com"
	}
	type req struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}
	type embObj struct {
		Embedding []float32 `json:"embedding"`
	}
	type resp struct {
		Data  []embObj `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	body, _ := json.Marshal(req{Input: texts, Model: e.Model()})
	r, err := httpPost(ctx, e.httpClient(), base+"/v1/embeddings",
		map[string]string{"Authorization": "Bearer " + e.APIKey}, body)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	var out resp
	if err := json.Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("openai embed decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("openai embed: %s", out.Error.Message)
	}
	result := make([][]float32, len(out.Data))
	for i, d := range out.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

func (e *OpenAIEmbedder) httpClient() *http.Client {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}
	return e.client
}

// ─── Google Gemini ────────────────────────────────────────────────────────────

// GeminiEmbedder uses gemini-embedding-001 (768 dims).
type GeminiEmbedder struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

func (e *GeminiEmbedder) Name() string  { return "gemini" }
func (e *GeminiEmbedder) Model() string { return "gemini-embedding-001" }
func (e *GeminiEmbedder) Dims() int     { return 768 }

func (e *GeminiEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base := e.BaseURL
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	// Gemini batch embed
	type part struct {
		Text string `json:"text"`
	}
	type reqItem struct {
		Model   string `json:"model"`
		Content struct {
			Parts []part `json:"parts"`
		} `json:"content"`
	}
	type batchReq struct {
		Requests []reqItem `json:"requests"`
	}
	type embObj struct {
		Values []float32 `json:"values"`
	}
	type resp struct {
		Embeddings []embObj `json:"embeddings"`
		Error      *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	items := make([]reqItem, len(texts))
	for i, t := range texts {
		items[i] = reqItem{Model: "models/" + e.Model()}
		items[i].Content.Parts = []part{{Text: t}}
	}
	body, _ := json.Marshal(batchReq{Requests: items})
	url := fmt.Sprintf("%s/v1beta/models/%s:batchEmbedContents?key=%s", base, e.Model(), e.APIKey)
	r, err := httpPost(ctx, e.httpClient(), url, nil, body)
	if err != nil {
		return nil, fmt.Errorf("gemini embed: %w", err)
	}
	var out resp
	if err := json.Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("gemini embed decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("gemini embed: %s", out.Error.Message)
	}
	result := make([][]float32, len(out.Embeddings))
	for i, e := range out.Embeddings {
		result[i] = e.Values
	}
	return result, nil
}

func (e *GeminiEmbedder) httpClient() *http.Client {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}
	return e.client
}

// ─── Voyage AI ────────────────────────────────────────────────────────────────

// VoyageEmbedder uses voyage-3-large (1024 dims).
type VoyageEmbedder struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

func (e *VoyageEmbedder) Name() string  { return "voyage" }
func (e *VoyageEmbedder) Model() string { return "voyage-3-large" }
func (e *VoyageEmbedder) Dims() int     { return 1024 }

func (e *VoyageEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base := e.BaseURL
	if base == "" {
		base = "https://api.voyageai.com"
	}
	type req struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}
	type embObj struct {
		Embedding []float32 `json:"embedding"`
	}
	type resp struct {
		Data  []embObj `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	body, _ := json.Marshal(req{Input: texts, Model: e.Model()})
	r, err := httpPost(ctx, e.httpClient(), base+"/v1/embeddings",
		map[string]string{"Authorization": "Bearer " + e.APIKey}, body)
	if err != nil {
		return nil, fmt.Errorf("voyage embed: %w", err)
	}
	var out resp
	if err := json.Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("voyage embed decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("voyage embed: %s", out.Error.Message)
	}
	result := make([][]float32, len(out.Data))
	for i, d := range out.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

func (e *VoyageEmbedder) httpClient() *http.Client {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}
	return e.client
}

// ─── Mistral ──────────────────────────────────────────────────────────────────

// MistralEmbedder uses mistral-embed (1024 dims).
type MistralEmbedder struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

func (e *MistralEmbedder) Name() string  { return "mistral" }
func (e *MistralEmbedder) Model() string { return "mistral-embed" }
func (e *MistralEmbedder) Dims() int     { return 1024 }

func (e *MistralEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base := e.BaseURL
	if base == "" {
		base = "https://api.mistral.ai"
	}
	type req struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}
	type embObj struct {
		Embedding []float32 `json:"embedding"`
	}
	type resp struct {
		Data  []embObj `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	body, _ := json.Marshal(req{Input: texts, Model: e.Model()})
	r, err := httpPost(ctx, e.httpClient(), base+"/v1/embeddings",
		map[string]string{"Authorization": "Bearer " + e.APIKey}, body)
	if err != nil {
		return nil, fmt.Errorf("mistral embed: %w", err)
	}
	var out resp
	if err := json.Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("mistral embed decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("mistral embed: %s", out.Error.Message)
	}
	result := make([][]float32, len(out.Data))
	for i, d := range out.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

func (e *MistralEmbedder) httpClient() *http.Client {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}
	return e.client
}

// ─── Ollama ───────────────────────────────────────────────────────────────────

// OllamaEmbedder uses nomic-embed-text via a local Ollama instance (768 dims).
type OllamaEmbedder struct {
	BaseURL string // default http://localhost:11434
	client  *http.Client
}

func (e *OllamaEmbedder) Name() string  { return "ollama" }
func (e *OllamaEmbedder) Model() string { return "nomic-embed-text" }
func (e *OllamaEmbedder) Dims() int     { return 768 }

func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	base := e.BaseURL
	if base == "" {
		base = "http://localhost:11434"
	}
	type req struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	type resp struct {
		Embedding []float32 `json:"embedding"`
		Error     string    `json:"error,omitempty"`
	}

	result := make([][]float32, len(texts))
	for i, text := range texts {
		body, _ := json.Marshal(req{Model: e.Model(), Prompt: text})
		r, err := httpPost(ctx, e.httpClient(), base+"/api/embeddings", nil, body)
		if err != nil {
			return nil, fmt.Errorf("ollama embed: %w", err)
		}
		var out resp
		if err := json.Unmarshal(r, &out); err != nil {
			return nil, fmt.Errorf("ollama embed decode: %w", err)
		}
		if out.Error != "" {
			return nil, fmt.Errorf("ollama embed: %s", out.Error)
		}
		result[i] = out.Embedding
	}
	return result, nil
}

func (e *OllamaEmbedder) httpClient() *http.Client {
	if e.client == nil {
		e.client = &http.Client{Timeout: 60 * time.Second}
	}
	return e.client
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func httpPost(ctx context.Context, c *http.Client, url string, headers map[string]string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	return data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
