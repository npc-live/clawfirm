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
// Three orthogonal axes: Platform × Protocol × Model.
//
//	Platform: where the API is hosted (zenmux, openrouter, google, openai)
//	Protocol: wire format       (google = Vertex AI generateContent,
//	                              openai = /images/generations,
//	                              openai-chat = /chat/completions + modalities)
//	Model:    model ID          (openai/gpt-image-2, google/gemini-3.1-flash-image-preview, …)
type MediaGen struct {
	// Platform is the hosting service: "zenmux", "openrouter", "openai", "google".
	Platform string
	// Protocol is the API wire format: "google", "openai", "openai-chat".
	// Auto-inferred from Platform if empty.
	Protocol string
	// Model is the model ID. Defaults depend on platform.
	Model string
	// APIKey is injected directly — shared with the media_understand provider.
	APIKey string
	// BaseURL overrides the platform's default endpoint.
	BaseURL string
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

	platform := m.Platform
	protocol := m.Protocol
	apiKey := m.APIKey
	baseURL := m.BaseURL

	// Auto-detect platform + key from env when not configured.
	if apiKey == "" {
		switch {
		case os.Getenv("ZENMUX_API_KEY") != "":
			apiKey = os.Getenv("ZENMUX_API_KEY")
			if platform == "" {
				platform = "zenmux"
			}
		case os.Getenv("GEMINI_API_KEY") != "":
			apiKey = os.Getenv("GEMINI_API_KEY")
			if platform == "" {
				platform = "google"
			}
		case os.Getenv("GOOGLE_API_KEY") != "":
			apiKey = os.Getenv("GOOGLE_API_KEY")
			if platform == "" {
				platform = "google"
			}
		case os.Getenv("OPENAI_API_KEY") != "":
			apiKey = os.Getenv("OPENAI_API_KEY")
			if platform == "" {
				platform = "openai"
			}
		case os.Getenv("OPENROUTER_API_KEY") != "":
			apiKey = os.Getenv("OPENROUTER_API_KEY")
			if platform == "" {
				platform = "openrouter"
			}
		case os.Getenv("OPENROUTER_APIKEY") != "":
			apiKey = os.Getenv("OPENROUTER_APIKEY")
			if platform == "" {
				platform = "openrouter"
			}
		}
	}

	// Detect platform from base URL when config passes an LLM protocol type
	// (e.g. "anthropic") as the platform name.
	if platform != "" {
		switch platform {
		case "zenmux", "openrouter", "openai", "google":
			// known platform
		default:
			// Not a recognized platform — try to infer from base URL.
			switch {
			case strings.Contains(baseURL, "zenmux.ai"):
				platform = "zenmux"
			case strings.Contains(baseURL, "openrouter.ai"):
				platform = "openrouter"
			case strings.Contains(baseURL, "openai.com"):
				platform = "openai"
			default:
				platform = "google"
			}
		}
	}
	if platform == "" {
		platform = "google"
	}

	if apiKey == "" {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{
				Type: types.ContentTypeText,
				Text: "media_gen: no API key found. Set ZENMUX_API_KEY, GEMINI_API_KEY, or OPENAI_API_KEY, or configure tools.media_gen in config.yml.",
			}},
		}, nil
	}

	// Infer protocol from platform when not explicitly set.
	if protocol == "" {
		switch platform {
		case "zenmux":
			protocol = "google"
		case "openrouter":
			protocol = "openai-chat"
		case "openai":
			protocol = "openai"
		default:
			protocol = "google"
		}
	}

	// Resolve base URL: platform-specific defaults + endpoint normalization.
	if baseURL == "" {
		baseURL = os.Getenv("OPENROUTER_BASE_URL")
	}
	if baseURL == "" {
		switch platform {
		case "zenmux":
			baseURL = "https://zenmux.ai/api/vertex-ai"
		case "openrouter":
			baseURL = "https://openrouter.ai/api/v1"
		}
	}
	if platform == "zenmux" && !strings.Contains(baseURL, "/api/vertex-ai") {
		if i := strings.Index(baseURL, "/api/"); i >= 0 {
			baseURL = baseURL[:i] + "/api/vertex-ai"
		} else {
			baseURL = strings.TrimRight(baseURL, "/") + "/api/vertex-ai"
		}
	}

	// Resolve model: config > env > platform default.
	model := m.Model
	if model == "" {
		if envModel := os.Getenv("CLAWFIRM_IMAGE_GEN_MODEL"); envModel != "" {
			model = envModel
		} else {
			switch platform {
			case "openai":
				model = "gpt-image-1"
			case "zenmux":
				model = "google/gemini-3.1-flash-image-preview"
			case "openrouter":
				model = "google/gemini-3.1-flash-image-preview"
			default:
				model = "gemini-2.5-flash-preview-04-17"
			}
		}
	}

	if onUpdate != nil {
		onUpdate(tool.ToolUpdate{Content: []types.ContentBlock{&types.TextContent{
			Type: types.ContentTypeText,
			Text: fmt.Sprintf("Generating image via %s [%s] model=%s size=%s…", platform, protocol, model, sizeKey),
		}}})
	}

	// Dispatch by protocol.
	// For Google protocol: use predict endpoint for dedicated image-gen models
	// (gpt-image-*, imagen-*), generateContent for Gemini models.
	var (
		imgData []byte
		err     error
	)
	switch protocol {
	case "openai":
		imgData, err = generateOpenAIImage(ctx, apiKey, model, prompt, sizeKey, quality, style)
	case "openai-chat":
		imgData, err = generateChatImage(ctx, apiKey, model, prompt, sizeKey, baseURL)
	default: // "google" — Vertex AI
		if isImageGenModel(model) {
			imgData, err = generateVertexImage(ctx, apiKey, model, prompt, sizeKey, baseURL)
		} else {
			imgData, err = generateGeminiImage(ctx, apiKey, model, prompt, sizeKey, baseURL)
		}
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

	result := fmt.Sprintf("Image saved to: %s\nPlatform: %s | Protocol: %s | Model: %s | Size: %s | Bytes: %d",
		outputPath, platform, protocol, model, sizeKey, len(imgData))

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
// When baseURL is non-empty, it targets a proxy (e.g. Zenmux Vertex AI) using Bearer auth.
func generateGeminiImage(ctx context.Context, apiKey, model, prompt, sizeKey, baseURL string) ([]byte, error) {
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
			{"role": "user", "parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"responseModalities": []string{"TEXT", "IMAGE"},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	var endpoint string
	var useBearer bool
	if baseURL != "" {
		endpoint = fmt.Sprintf("%s/v1/models/%s:generateContent",
			strings.TrimRight(baseURL, "/"), model)
		useBearer = true
	} else {
		endpoint = fmt.Sprintf(
			"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
			model, apiKey)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if useBearer {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

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

// isImageGenModel returns true for dedicated image generation models that use
// the Vertex AI predict endpoint instead of generateContent.
func isImageGenModel(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "gpt-image") ||
		strings.Contains(m, "imagen") ||
		strings.Contains(m, "dall-e")
}

// --- Vertex AI predict endpoint (for gpt-image-2, Imagen, etc.) ---

// generateVertexImage calls the Vertex AI models/:predict endpoint used by
// dedicated image generation models (e.g. openai/gpt-image-2 via Zenmux).
// This matches the Python SDK's client.models.generate_images() method.
func generateVertexImage(ctx context.Context, apiKey, model, prompt, sizeKey, baseURL string) ([]byte, error) {
	aspectMap := map[string]string{
		"portrait":      "9:16",
		"landscape":     "16:9",
		"landscape_4x3": "4:3",
		"square":        "1:1",
	}
	aspectRatio := aspectMap[sizeKey]
	if aspectRatio == "" {
		aspectRatio = "9:16"
	}

	params := map[string]any{
		"sampleCount": 1,
		"aspectRatio": aspectRatio,
	}

	reqBody := map[string]any{
		"instances":  []map[string]any{{"prompt": prompt}},
		"parameters": params,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("vertex-image: marshal request: %w", err)
	}

	if baseURL == "" {
		baseURL = "https://zenmux.ai/api/vertex-ai"
	}
	endpoint := fmt.Sprintf("%s/v1/models/%s:predict",
		strings.TrimRight(baseURL, "/"), model)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("vertex-image: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vertex-image: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vertex-image: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vertex-image: HTTP %d: %s", resp.StatusCode, string(raw[:min(500, len(raw))]))
	}

	var apiResp struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
			MimeType           string `json:"mimeType"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("vertex-image: parse response: %w", err)
	}
	if len(apiResp.Predictions) == 0 || apiResp.Predictions[0].BytesBase64Encoded == "" {
		return nil, fmt.Errorf("vertex-image: no image data in response: %s", string(raw[:min(300, len(raw))]))
	}

	imgData, err := base64.StdEncoding.DecodeString(apiResp.Predictions[0].BytesBase64Encoded)
	if err != nil {
		return nil, fmt.Errorf("vertex-image: base64 decode: %w", err)
	}
	return imgData, nil
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

// --- OpenAI-compatible chat completions image generation (Zenmux / OpenRouter) ---

// generateChatImage uses the /chat/completions endpoint with modalities=["image"],
// compatible with Zenmux and OpenRouter for models like gemini-3.1-flash-image-preview.
func generateChatImage(ctx context.Context, apiKey, model, prompt, sizeKey, baseURL string) ([]byte, error) {
	sizeHints := map[string]string{
		"portrait":      "Generate in vertical 9:16 portrait aspect ratio (e.g. 1080×1920).",
		"landscape":     "Generate in horizontal 16:9 landscape aspect ratio (e.g. 1920×1080).",
		"landscape_4x3": "Generate in horizontal 4:3 aspect ratio (e.g. 1440×1080).",
		"square":        "Generate in square 1:1 aspect ratio (e.g. 1024×1024).",
	}
	if hint, ok := sizeHints[sizeKey]; ok {
		prompt = prompt + "\n\n" + hint
	}

	body := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
		"modalities": []string{"image"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("chat_image: marshal: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("chat_image: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat_image: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chat_image: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat_image: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
				Images  []struct {
					ImageURL struct{ URL string `json:"url"` } `json:"image_url"`
				} `json:"images"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("chat_image: parse response: %v", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("chat_image: empty choices in response")
	}
	msg := result.Choices[0].Message

	// Primary: message.images (OpenRouter/Zenmux Gemini image format)
	for _, img := range msg.Images {
		if u := img.ImageURL.URL; len(u) > 5 {
			return decodeImageDataURL(u)
		}
	}

	// Fallback: message.content as string data URL
	if s, ok := msg.Content.(string); ok && strings.HasPrefix(s, "data:") {
		return decodeImageDataURL(s)
	}

	// Fallback: content array
	if parts, ok := msg.Content.([]any); ok {
		for _, p := range parts {
			if m, ok := p.(map[string]any); ok {
				if iu, ok := m["image_url"].(map[string]any); ok {
					if u, ok := iu["url"].(string); ok && len(u) > 5 {
						return decodeImageDataURL(u)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("chat_image: no image found in response: %s", string(raw[:min(300, len(raw))]))
}

func decodeImageDataURL(u string) ([]byte, error) {
	idx := strings.IndexByte(u, ',')
	if idx < 0 {
		return nil, fmt.Errorf("invalid image data URL")
	}
	return base64.StdEncoding.DecodeString(u[idx+1:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
