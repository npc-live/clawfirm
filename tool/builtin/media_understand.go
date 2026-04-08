package builtin

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
)

const mediaMaxFileSize = 50 * 1024 * 1024 // 50 MB

// MediaUnderstand uses a dedicated multimodal LLM to analyze images and videos.
type MediaUnderstand struct {
	Provider provider.LLMProvider // injected multimodal provider
	Model    string               // model ID
}

func (m *MediaUnderstand) Name() string  { return "media_understand" }
func (m *MediaUnderstand) Label() string { return "Media Understand" }
func (m *MediaUnderstand) Description() string {
	return "Analyze an image or video using a multimodal LLM. Provide a local file path or URL, plus a prompt describing what to analyze."
}

func (m *MediaUnderstand) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "Local file path to an image or video.",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "Remote URL of an image or video.",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "What to analyze about the media.",
			},
		},
		"required": []string{"prompt"},
	}
}

func (m *MediaUnderstand) Execute(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
	if m.Provider == nil {
		return tool.ToolResult{
			Content: []types.ContentBlock{&types.TextContent{
				Type: types.ContentTypeText,
				Text: "media_understand is not configured. Add a `media` section to config.yml with a multimodal provider and model.",
			}},
		}, nil
	}

	prompt, _ := params["prompt"].(string)
	if prompt == "" {
		return tool.ToolResult{}, fmt.Errorf("media_understand: prompt is required")
	}

	filePath, _ := params["file"].(string)
	urlStr, _ := params["url"].(string)

	if filePath == "" && urlStr == "" {
		return tool.ToolResult{}, fmt.Errorf("media_understand: at least one of file or url is required")
	}

	// Build content blocks for the user message.
	var contentBlocks []types.ContentBlock

	if filePath != "" {
		block, err := mediaBlockFromFile(filePath)
		if err != nil {
			return tool.ToolResult{}, fmt.Errorf("media_understand: %w", err)
		}
		contentBlocks = append(contentBlocks, block)
	}

	if urlStr != "" {
		block := mediaBlockFromURL(urlStr)
		contentBlocks = append(contentBlocks, block)
	}

	// Append the text prompt.
	contentBlocks = append(contentBlocks, &types.TextContent{
		Type: types.ContentTypeText,
		Text: prompt,
	})

	// Build LLM request with single user message.
	req := provider.LLMRequest{
		Model: types.Model{
			ID:        m.Model,
			MaxTokens: 4096,
		},
		Messages: []types.Message{
			types.NewUserMessage(contentBlocks...),
		},
	}

	ch, err := m.Provider.Stream(ctx, req)
	if err != nil {
		return tool.ToolResult{}, fmt.Errorf("media_understand: stream: %w", err)
	}

	// Collect all text deltas.
	var sb strings.Builder
	for ev := range ch {
		switch ev.Type {
		case types.StreamEventTextDelta:
			sb.WriteString(ev.Delta)
			if onUpdate != nil {
				onUpdate(tool.ToolUpdate{
					Content: []types.ContentBlock{&types.TextContent{
						Type: types.ContentTypeText,
						Text: sb.String(),
					}},
				})
			}
		case types.StreamEventError:
			errMsg := "unknown error"
			if ev.Error != nil {
				errMsg = ev.Error.ErrorMessage
			}
			return tool.ToolResult{}, fmt.Errorf("media_understand: provider error: %s", errMsg)
		}
	}

	return tool.ToolResult{
		Content: []types.ContentBlock{&types.TextContent{
			Type: types.ContentTypeText,
			Text: sb.String(),
		}},
	}, nil
}

// mediaBlockFromFile reads a local file and returns an ImageContent or VideoContent block.
func mediaBlockFromFile(path string) (types.ContentBlock, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	if info.Size() > mediaMaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (limit %d)", info.Size(), mediaMaxFileSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	mimeType := mimeFromExt(filepath.Ext(path))
	encoded := base64.StdEncoding.EncodeToString(data)

	if isVideoMime(mimeType) {
		return &types.VideoContent{
			Type:     types.ContentTypeVideo,
			Data:     encoded,
			MimeType: mimeType,
		}, nil
	}
	return &types.ImageContent{
		Type:     types.ContentTypeImage,
		Data:     encoded,
		MimeType: mimeType,
	}, nil
}

// mediaBlockFromURL returns an ImageContent or VideoContent block for a remote URL.
func mediaBlockFromURL(url string) types.ContentBlock {
	ext := filepath.Ext(url)
	// Strip query string from extension.
	if idx := strings.IndexByte(ext, '?'); idx != -1 {
		ext = ext[:idx]
	}
	mimeType := mimeFromExt(ext)

	if isVideoMime(mimeType) {
		return &types.VideoContent{
			Type:     types.ContentTypeVideo,
			URL:      url,
			MimeType: mimeType,
		}
	}
	return &types.ImageContent{
		Type:     types.ContentTypeImage,
		URL:      url,
		MimeType: mimeType,
	}
}

// mimeFromExt returns the MIME type for a file extension, defaulting to image/png.
func mimeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	// Common overrides for reliable detection.
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	}
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	return "image/png"
}

// isVideoMime returns true if the MIME type represents a video format.
func isVideoMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}
