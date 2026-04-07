// Command media-gen is a claw-code plugin tool.
// It generates an image from a text prompt using a generative LLM and saves it to a file.
//
// Input (CLAWD_TOOL_INPUT env, JSON):
//
//	{"prompt": "...", "output_path": "/path/to/output.png"}
//
// Output (stdout): path of the saved image.
//
// API keys (first match wins):
//
//	OPENROUTER_API_KEY / OPENROUTER_APIKEY → OpenRouter (google/gemini-3.1-flash-image-preview)
//	GEMINI_API_KEY / GOOGLE_API_KEY        → Google Generative AI native API
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type input struct {
	Prompt     string `json:"prompt"`
	OutputPath string `json:"output_path"`
}

func main() {
	raw := os.Getenv("CLAWD_TOOL_INPUT")
	if raw == "" {
		fmt.Fprintln(os.Stderr, "error: CLAWD_TOOL_INPUT not set")
		os.Exit(1)
	}
	var inp input
	if err := json.Unmarshal([]byte(raw), &inp); err != nil {
		fmt.Fprintf(os.Stderr, "error: bad input: %v\n", err)
		os.Exit(1)
	}
	if inp.Prompt == "" {
		fmt.Fprintln(os.Stderr, "error: prompt is required")
		os.Exit(1)
	}
	if inp.OutputPath == "" {
		inp.OutputPath = "/tmp/media_gen_output.png"
	}
	if err := os.MkdirAll(filepath.Dir(inp.OutputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: mkdir: %v\n", err)
		os.Exit(1)
	}

	orKey := firstEnv("OPENROUTER_API_KEY", "OPENROUTER_APIKEY")
	geminiKey := firstEnv("GEMINI_API_KEY", "GOOGLE_API_KEY")

	var imgBytes []byte
	var err error
	if orKey != "" {
		imgBytes, err = genOpenRouter(orKey, inp.Prompt)
	} else if geminiKey != "" {
		imgBytes, err = genGemini(geminiKey, inp.Prompt)
	} else {
		fmt.Fprintln(os.Stderr, "error: set OPENROUTER_API_KEY or GEMINI_API_KEY")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(inp.OutputPath, imgBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write image: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(inp.OutputPath)
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func doPost(url string, headers map[string]string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 190 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, respBody)
	}
	return respBody, nil
}

func genOpenRouter(apiKey, prompt string) ([]byte, error) {
	model := os.Getenv("CLAWFIRM_IMAGE_GEN_MODEL")
	if model == "" {
		model = "google/gemini-3.1-flash-image-preview"
	}
	body := map[string]any{
		"model":      model,
		"messages":   []map[string]any{{"role": "user", "content": prompt}},
		"modalities": []string{"image"},
	}
	respBytes, err := doPost("https://openrouter.ai/api/v1/chat/completions",
		map[string]string{"Authorization": "Bearer " + apiKey}, body)
	if err != nil {
		return nil, err
	}

	// OpenRouter Gemini image gen returns image in message.images[]
	var result struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
				Images  []struct {
					ImageURL struct {
						URL string `json:"url"`
					} `json:"image_url"`
				} `json:"images"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("parse response: %v", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty choices")
	}
	msg := result.Choices[0].Message

	// Primary: message.images
	for _, img := range msg.Images {
		if u := img.ImageURL.URL; len(u) > 5 {
			return decodeDataURL(u)
		}
	}

	// Fallback: message.content as string data URL
	if s, ok := msg.Content.(string); ok && len(s) > 5 {
		return decodeDataURL(s)
	}

	return nil, fmt.Errorf("no image in response: %s", respBytes[:min(200, len(respBytes))])
}

func genGemini(apiKey, prompt string) ([]byte, error) {
	model := os.Getenv("CLAWFIRM_IMAGE_GEN_MODEL")
	if model == "" {
		model = "gemini-3.1-flash-image-preview"
	}
	body := map[string]any{
		"contents": []map[string]any{{"parts": []map[string]any{{"text": prompt}}}},
		"generationConfig": map[string]any{
			"responseModalities": []string{"IMAGE", "TEXT"},
		},
	}
	baseURL := os.Getenv("GEMINI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	} else if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/v1beta/models/" + model + ":generateContent?key=" + apiKey
	respBytes, err := doPost(url, nil, body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData *struct {
						Data string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("parse response: %v", err)
	}
	for _, c := range result.Candidates {
		for _, p := range c.Content.Parts {
			if p.InlineData != nil {
				return base64.StdEncoding.DecodeString(p.InlineData.Data)
			}
		}
	}
	return nil, fmt.Errorf("no image in response")
}

func decodeDataURL(u string) ([]byte, error) {
	// format: data:<mime>;base64,<data>
	for i := 0; i < len(u); i++ {
		if u[i] == ',' {
			return base64.StdEncoding.DecodeString(u[i+1:])
		}
	}
	return nil, fmt.Errorf("invalid data URL")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
