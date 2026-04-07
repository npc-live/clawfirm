// Command media-understand is a claw-code plugin tool.
// It analyses an image file using a vision LLM and writes a text description to stdout.
//
// Input (CLAWD_TOOL_INPUT env, JSON):
//
//	{"file_path": "/path/to/image.jpg", "prompt": "optional prompt"}
//
// Output (stdout): text description of the image.
//
// API keys (first match wins):
//
//	OPENROUTER_API_KEY / OPENROUTER_APIKEY → OpenRouter (google/gemini-2.5-flash)
//	GEMINI_API_KEY / GOOGLE_API_KEY        → Google Generative AI native API
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type input struct {
	FilePath string `json:"file_path"`
	Prompt   string `json:"prompt"`
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
		inp.Prompt = "请详细描述这张图片或视频帧的内容。"
	}

	data, err := os.ReadFile(inp.FilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read file: %v\n", err)
		os.Exit(1)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	mimeType := mime.TypeByExtension(filepath.Ext(inp.FilePath))
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	orKey := firstEnv("OPENROUTER_API_KEY", "OPENROUTER_APIKEY")
	geminiKey := firstEnv("GEMINI_API_KEY", "GOOGLE_API_KEY")

	var text string
	if orKey != "" {
		text, err = callOpenRouter(orKey, b64, mimeType, inp.Prompt)
	} else if geminiKey != "" {
		text, err = callGeminiVision(geminiKey, b64, mimeType, inp.Prompt)
	} else {
		fmt.Fprintln(os.Stderr, "error: set OPENROUTER_API_KEY or GEMINI_API_KEY")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(text)
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

func callOpenRouter(apiKey, b64, mimeType, prompt string) (string, error) {
	model := os.Getenv("CLAWFIRM_MEDIA_MODEL")
	if model == "" {
		model = "google/gemini-2.5-flash"
	}
	body := map[string]any{
		"model": model,
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{"type": "image_url", "image_url": map[string]any{
					"url": "data:" + mimeType + ";base64," + b64,
				}},
				{"type": "text", "text": prompt},
			},
		}},
	}
	respBytes, err := doPost("https://openrouter.ai/api/v1/chat/completions",
		map[string]string{"Authorization": "Bearer " + apiKey}, body)
	if err != nil {
		return "", err
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %v", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty choices in response")
	}
	return result.Choices[0].Message.Content, nil
}

func callGeminiVision(apiKey, b64, mimeType, prompt string) (string, error) {
	model := os.Getenv("CLAWFIRM_MEDIA_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}
	body := map[string]any{
		"contents": []map[string]any{{
			"parts": []map[string]any{
				{"inline_data": map[string]any{"mime_type": mimeType, "data": b64}},
				{"text": prompt},
			},
		}},
	}
	baseURL := os.Getenv("GEMINI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	url := baseURL + "/v1beta/models/" + model + ":generateContent?key=" + apiKey
	respBytes, err := doPost(url, nil, body)
	if err != nil {
		return "", err
	}
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %v", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}
