package types

import (
	"encoding/json"
	"testing"
)

func TestUserMessageRoundTrip(t *testing.T) {
	orig := &UserMessage{
		Role: "user",
		Content: []ContentBlock{
			&TextContent{Type: ContentTypeText, Text: "hello"},
			&ImageContent{Type: ContentTypeImage, Data: "abc123", MimeType: "image/jpeg"},
		},
		Timestamp: 1700000000,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got UserMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Role != "user" {
		t.Errorf("Role: got %q want %q", got.Role, "user")
	}
	if len(got.Content) != 2 {
		t.Fatalf("Content len: got %d want 2", len(got.Content))
	}
	tc, ok := got.Content[0].(*TextContent)
	if !ok {
		t.Fatalf("Content[0] not *TextContent, got %T", got.Content[0])
	}
	if tc.Text != "hello" {
		t.Errorf("TextContent.Text: got %q want %q", tc.Text, "hello")
	}
	ic, ok := got.Content[1].(*ImageContent)
	if !ok {
		t.Fatalf("Content[1] not *ImageContent, got %T", got.Content[1])
	}
	if ic.Data != "abc123" {
		t.Errorf("ImageContent.Data: got %q want %q", ic.Data, "abc123")
	}
	if got.Timestamp != 1700000000 {
		t.Errorf("Timestamp: got %d want %d", got.Timestamp, 1700000000)
	}
}

func TestAssistantMessageWithToolCallRoundTrip(t *testing.T) {
	orig := &AssistantMessage{
		Role: "assistant",
		Content: []ContentBlock{
			&TextContent{Type: ContentTypeText, Text: "calling tool"},
			&ToolCall{
				Type:      ContentTypeToolCall,
				ID:        "call_1",
				Name:      "search",
				Arguments: map[string]any{"q": "go testing"},
			},
		},
		Provider:   "anthropic",
		Model:      "claude-opus-4-6",
		StopReason: StopReasonToolUse,
		Timestamp:  1700000001,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AssistantMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Role != "assistant" {
		t.Errorf("Role: got %q want %q", got.Role, "assistant")
	}
	if len(got.Content) != 2 {
		t.Fatalf("Content len: got %d want 2", len(got.Content))
	}
	tc2, ok := got.Content[1].(*ToolCall)
	if !ok {
		t.Fatalf("Content[1] not *ToolCall, got %T", got.Content[1])
	}
	if tc2.ID != "call_1" {
		t.Errorf("ToolCall.ID: got %q want %q", tc2.ID, "call_1")
	}
	if got.StopReason != StopReasonToolUse {
		t.Errorf("StopReason: got %q want %q", got.StopReason, StopReasonToolUse)
	}
}

func TestToolResultMessageRoundTrip(t *testing.T) {
	orig := &ToolResultMessage{
		Role:       "tool",
		ToolCallID: "call_1",
		ToolName:   "search",
		Content: []ContentBlock{
			&TextContent{Type: ContentTypeText, Text: "result text"},
		},
		IsError:   false,
		Timestamp: 1700000002,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ToolResultMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ToolCallID != "call_1" {
		t.Errorf("ToolCallID: got %q want %q", got.ToolCallID, "call_1")
	}
	if got.ToolName != "search" {
		t.Errorf("ToolName: got %q want %q", got.ToolName, "search")
	}
	if len(got.Content) != 1 {
		t.Fatalf("Content len: got %d want 1", len(got.Content))
	}
	tc, ok := got.Content[0].(*TextContent)
	if !ok {
		t.Fatalf("Content[0] not *TextContent, got %T", got.Content[0])
	}
	if tc.Text != "result text" {
		t.Errorf("TextContent.Text: got %q want %q", tc.Text, "result text")
	}
}

func TestMessageInterface(t *testing.T) {
	var _ Message = &UserMessage{}
	var _ Message = &AssistantMessage{}
	var _ Message = &ToolResultMessage{}
}

func TestMessageRoles(t *testing.T) {
	u := &UserMessage{}
	if u.MessageRole() != "user" {
		t.Errorf("UserMessage role: got %q want %q", u.MessageRole(), "user")
	}
	a := &AssistantMessage{}
	if a.MessageRole() != "assistant" {
		t.Errorf("AssistantMessage role: got %q want %q", a.MessageRole(), "assistant")
	}
	tr := &ToolResultMessage{}
	if tr.MessageRole() != "tool" {
		t.Errorf("ToolResultMessage role: got %q want %q", tr.MessageRole(), "tool")
	}
}

func TestUnmarshalUnknownContentType(t *testing.T) {
	jsonStr := `{"role":"user","content":[{"type":"unknown","value":"x"}],"timestamp":0}`
	var m UserMessage
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err == nil {
		t.Fatal("expected error for unknown content type, got nil")
	}
}
