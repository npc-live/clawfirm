package types

import (
	"encoding/json"
	"testing"
)

func TestTextContentRoundTrip(t *testing.T) {
	orig := &TextContent{
		Type:          ContentTypeText,
		Text:          "hello world",
		TextSignature: "sig123",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got TextContent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Text != orig.Text {
		t.Errorf("Text: got %q want %q", got.Text, orig.Text)
	}
	if got.TextSignature != orig.TextSignature {
		t.Errorf("TextSignature: got %q want %q", got.TextSignature, orig.TextSignature)
	}
	if got.ContentType() != ContentTypeText {
		t.Errorf("ContentType: got %q want %q", got.ContentType(), ContentTypeText)
	}
}

func TestImageContentRoundTrip(t *testing.T) {
	orig := &ImageContent{
		Type:     ContentTypeImage,
		Data:     "base64data",
		MimeType: "image/png",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ImageContent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Data != orig.Data {
		t.Errorf("Data: got %q want %q", got.Data, orig.Data)
	}
	if got.MimeType != orig.MimeType {
		t.Errorf("MimeType: got %q want %q", got.MimeType, orig.MimeType)
	}
	if got.ContentType() != ContentTypeImage {
		t.Errorf("ContentType: got %q want %q", got.ContentType(), ContentTypeImage)
	}
}

func TestAudioContentRoundTrip(t *testing.T) {
	orig := &AudioContent{
		Type:     ContentTypeAudio,
		URL:      "https://example.com/audio.mp3",
		MimeType: "audio/mp3",
		Duration: 3.5,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AudioContent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.URL != orig.URL {
		t.Errorf("URL: got %q want %q", got.URL, orig.URL)
	}
	if got.Duration != orig.Duration {
		t.Errorf("Duration: got %v want %v", got.Duration, orig.Duration)
	}
	if got.ContentType() != ContentTypeAudio {
		t.Errorf("ContentType: got %q want %q", got.ContentType(), ContentTypeAudio)
	}
}

func TestToolCallRoundTrip(t *testing.T) {
	orig := &ToolCall{
		Type:      ContentTypeToolCall,
		ID:        "call_abc",
		Name:      "search",
		Arguments: map[string]any{"query": "golang"},
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ToolCall
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != orig.ID {
		t.Errorf("ID: got %q want %q", got.ID, orig.ID)
	}
	if got.Name != orig.Name {
		t.Errorf("Name: got %q want %q", got.Name, orig.Name)
	}
	q, _ := got.Arguments["query"].(string)
	if q != "golang" {
		t.Errorf("Arguments.query: got %q want %q", q, "golang")
	}
	if got.ContentType() != ContentTypeToolCall {
		t.Errorf("ContentType: got %q want %q", got.ContentType(), ContentTypeToolCall)
	}
}

func TestThinkingContentRoundTrip(t *testing.T) {
	orig := &ThinkingContent{
		Type:              ContentTypeThinking,
		Thinking:          "let me think...",
		ThinkingSignature: "tsig",
		Redacted:          true,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ThinkingContent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Thinking != orig.Thinking {
		t.Errorf("Thinking: got %q want %q", got.Thinking, orig.Thinking)
	}
	if !got.Redacted {
		t.Errorf("Redacted: expected true")
	}
	if got.ContentType() != ContentTypeThinking {
		t.Errorf("ContentType: got %q want %q", got.ContentType(), ContentTypeThinking)
	}
}

func TestContentBlockInterface(t *testing.T) {
	// Ensure all types satisfy the interface
	var _ ContentBlock = &TextContent{}
	var _ ContentBlock = &ImageContent{}
	var _ ContentBlock = &AudioContent{}
	var _ ContentBlock = &ToolCall{}
	var _ ContentBlock = &ThinkingContent{}
}
