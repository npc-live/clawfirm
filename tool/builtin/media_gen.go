package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

// MediaGen generates images via an image-generation API and saves them to disk.
// Supports Gemini (gemini-2.5-flash-image, default) and OpenAI (dall-e-3).
type MediaGen struct {
	// Provider is the image generation backend: "gemini" (default) or "openai".
	Provider string
	// Model is the model ID, e.g. "gemini-2.5-flash-preview-04-17" or "dall-e-3".
	Model string
	// APIKey is injected directly — shared with the media_understand provider.
	APIKey string
}

func (m *MediaGen) Name() string  { return "media_gen" }
func (m *MediaGen) Label() string { return "Media Gen" }
func (m *MediaGen) Description() string {
	return "Generate an image from a text prompt using an AI image model (default: Gemini). " +
		"Saves the image to disk and returns the file path. " +
		"Use this to create cover images, thumbnails, or any visual asset."
}

func (m *MediaGen) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Detailed description of the image to generate.",
			},
			"size": map[string]any{
				"type":        "string",
				"enum":        []string{"portrait", "portrait_3x4", "landscape", "landscape_4x3", "square"},
				"description": "Output aspect ratio. portrait=9:16 (TikTok竖屏), portrait_3x4=3:4 (抖音/小红书竖版封面), landscape=16:9 (横屏封面), landscape_4x3=4:3 (通用横版, 默认必出), square=1:1. Default: portrait.",
			},
			"style": map[string]any{
				"type":        "string",
				"enum":        []string{"vivid", "natural"},
				"description": "vivid = hyper-real/dramatic (default), natural = realistic/muted. OpenAI only.",
			},
			"quality": map[string]any{
				"type":        "string",
				"enum":        []string{"standard", "hd"},
				"description": "Image quality for OpenAI: hd produces more detail. Default: standard.",
			},
			"output": map[string]any{
				"type":        "string",
				"description": "Output file path. Defaults to ~/Desktop/cover_TIMESTAMP.png.",
			},
		},
		"required": []string{"prompt"},
	}
}

func (m *MediaGen) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	prompt, _ := params["prompt"].(string)
	if prompt == "" {
		return tool.ToolResult{}, fmt.Errorf("media_gen: prompt is required")
	}

	sizeKey, _ := params["size"].(string)
	if sizeKey == "" {
		sizeKey = "portrait"
	}
	style, _ := params["style"].(string)
	if style == "" {
		style = "vivid"
	}
	quality, _ := params["quality"].(string)
	if quality == "" {
		quality = "standard"
	}
	outputPath, _ := params["output"].(string)
	if outputPath == "" {
		ts := time.Now().Format("20060102_150405")
		outputPath = fmt.Sprintf("~/Desktop/cover_%s.png", ts)
	}
	outputPath = expandHome(outputPath)

	prov := m.Provider
	if prov == "" {
		prov = "gemini"
	}
	model := m.Model
	if model == "" {
		switch prov {
		case "openai":
			model = "dall-e-3"
		default:
			model = "gemini-2.5-flash-preview-04-17"
		}
	}

	apiKey := m.APIKey
	if apiKey == "" {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{
				Type: types.ContentTypeText,
				Text: "media_gen is not configured. Add a `media` section to config.yml — media_gen shares its API key with media_understand.",
			}},
		}, nil
	}

	if onUpdate != nil {
		onUpdate(tool.ToolUpdate{Content: []types.ContentBlock{&types.TextContent{
			Type: types.ContentTypeText,
			Text: fmt.Sprintf("Generating image with %s/%s (%s)…", prov, model, sizeKey),
		}}})
	}

	var (
		imgData []byte
		err     error
	)
	switch prov {
	case "openai":
		imgData, err = generateOpenAIImage(ctx, apiKey, model, prompt, sizeKey, quality, style)
	default:
		imgData, err = generateGeminiImage(ctx, apiKey, model, prompt, sizeKey)
	}
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("media_gen: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return tool.ToolResult{}, fmt.Errorf("media_gen: mkdir: %w", err)
	}
	if err := os.WriteFile(outputPath, imgData, 0o644); err != nil {
		return tool.ToolResult{}, fmt.Errorf("media_gen: write: %w", err)
	}

	result := fmt.Sprintf("Image saved to: %s\nProvider: %s/%s | Size: %s | Bytes: %d",
		outputPath, prov, model, sizeKey, len(imgData))

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{
			Type: types.ContentTypeText,
			Text: result,
		}},
	}, nil
}

// --- Gemini image generation ---

// generateGeminiImage calls the Gemini generateContent API with responseModalities=IMAGE
// and returns the raw image bytes from the first inlineData part.
func generateGeminiImage(ctx context.Context, apiKey, model, prompt, sizeKey string) ([]byte, error) {
	// Aspect ratio hint appended to prompt since Gemini 2.x flash image gen
	// does not expose a dedicated size parameter via this endpoint.
	aspectHints := map[string]string{
		"portrait":      "vertical 9:16 aspect ratio",
		"portrait_3x4":  "vertical 3:4 aspect ratio",
		"landscape":     "horizontal 16:9 aspect ratio",
		"landscape_4x3": "horizontal 4:3 aspect ratio",
		"square":        "square 1:1 aspect ratio",
	}
	if hint, ok := aspectHints[sizeKey]; ok {
		prompt = prompt + " (" + hint + ")"
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"responseModalities": []string{"TEXT", "IMAGE"},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(raw, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("gemini API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("gemini API status %d: %s", resp.StatusCode, string(raw))
	}

	// Parse response — find the first inlineData part that is an image.
	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text"`
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"` // base64
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("gemini: parse response: %w", err)
	}

	for _, cand := range apiResp.Candidates {
		for _, part := range cand.Content.Parts {
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "image/") {
				imgData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("gemini: base64 decode: %w", err)
				}
				return imgData, nil
			}
		}
	}
	return nil, fmt.Errorf("gemini: no image data found in response")
}

// --- OpenAI DALL-E image generation ---

// generateOpenAIImage calls the OpenAI Images API and returns raw image bytes.
func generateOpenAIImage(ctx context.Context, apiKey, model, prompt, sizeKey, quality, style string) ([]byte, error) {
	sizeMap := map[string]string{
		"portrait":      "1024x1792",
		"portrait_3x4":  "1024x1365",
		"landscape":     "1792x1024",
		"landscape_4x3": "1440x1080",
		"square":        "1024x1024",
	}
	size, ok := sizeMap[sizeKey]
	if !ok {
		size = "1024x1792"
	}

	reqBody := map[string]any{
		"model":           model,
		"prompt":          prompt,
		"n":               1,
		"size":            size,
		"quality":         quality,
		"style":           style,
		"response_format": "b64_json",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/images/generations",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(raw, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("openai API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("openai API status %d: %s", resp.StatusCode, string(raw))
	}

	var apiResp struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("openai: parse response: %w", err)
	}
	if len(apiResp.Data) == 0 || apiResp.Data[0].B64JSON == "" {
		return nil, fmt.Errorf("openai: empty image data in response")
	}

	imgData, err := base64.StdEncoding.DecodeString(apiResp.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("openai: base64 decode: %w", err)
	}
	return imgData, nil
}
