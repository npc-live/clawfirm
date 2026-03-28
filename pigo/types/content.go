package types

// ContentType constants identify the kind of content in a ContentBlock.
const (
	ContentTypeText     = "text"
	ContentTypeImage    = "image"
	ContentTypeAudio    = "audio"
	ContentTypeToolCall = "toolCall"
	ContentTypeThinking = "thinking"
)

// ContentBlock is implemented by all content block types.
type ContentBlock interface {
	ContentType() string
}

// TextContent holds a plain-text or signature-bearing text block.
type TextContent struct {
	Type          string `json:"type"`
	Text          string `json:"text"`
	TextSignature string `json:"textSignature,omitempty"`
}

// ContentType returns the type identifier for TextContent.
func (t *TextContent) ContentType() string { return ContentTypeText }

// ImageContent holds an image, either as base64 data or a URL.
type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"` // base64-encoded image bytes
	URL      string `json:"url,omitempty"`
	MimeType string `json:"mimeType"`
}

// ContentType returns the type identifier for ImageContent.
func (i *ImageContent) ContentType() string { return ContentTypeImage }

// AudioContent holds audio data, either as base64 data or a URL.
type AudioContent struct {
	Type     string  `json:"type"`
	Data     string  `json:"data,omitempty"`
	URL      string  `json:"url,omitempty"`
	MimeType string  `json:"mimeType"`
	Duration float64 `json:"duration,omitempty"`
}

// ContentType returns the type identifier for AudioContent.
func (a *AudioContent) ContentType() string { return ContentTypeAudio }

// ToolCall represents a request to invoke a tool, emitted by the assistant.
type ToolCall struct {
	Type             string         `json:"type"`
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Arguments        map[string]any `json:"arguments"`
	ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}

// ContentType returns the type identifier for ToolCall.
func (tc *ToolCall) ContentType() string { return ContentTypeToolCall }

// ThinkingContent holds a chain-of-thought block produced by reasoning models.
type ThinkingContent struct {
	Type              string `json:"type"`
	Thinking          string `json:"thinking"`
	ThinkingSignature string `json:"thinkingSignature,omitempty"`
	Redacted          bool   `json:"redacted,omitempty"`
}

// ContentType returns the type identifier for ThinkingContent.
func (th *ThinkingContent) ContentType() string { return ContentTypeThinking }
