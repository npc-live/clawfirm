package agent

import (
	"strings"
	"testing"

	"github.com/ai-gateway/clawfirm/types"
)

func TestMicrocompactMessages(t *testing.T) {
	// Build a conversation with 5 turns: user → assistant → tool_result × 5
	msgs := []types.Message{
		// Turn 1
		&types.UserMessage{Role: "user"},
		&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
			&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c1", Name: "read"},
		}, StopReason: types.StopReasonToolUse},
		&types.ToolResultMessage{Role: "tool", ToolCallID: "c1", ToolName: "read", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "file content turn 1"},
		}},
		// Turn 2
		&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
			&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c2", Name: "bash"},
		}, StopReason: types.StopReasonToolUse},
		&types.ToolResultMessage{Role: "tool", ToolCallID: "c2", ToolName: "bash", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "bash output turn 2"},
		}},
		// Turn 3
		&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
			&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c3", Name: "edit"},
		}, StopReason: types.StopReasonToolUse},
		&types.ToolResultMessage{Role: "tool", ToolCallID: "c3", ToolName: "edit", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "edit result turn 3"},
		}},
		// Turn 4
		&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
			&types.ToolCall{Type: types.ContentTypeToolCall, ID: "c4", Name: "grep"},
		}, StopReason: types.StopReasonToolUse},
		&types.ToolResultMessage{Role: "tool", ToolCallID: "c4", ToolName: "grep", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "grep result turn 4"},
		}},
		// Turn 5 (final)
		&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "done"},
		}, StopReason: types.StopReasonStop},
	}

	result := MicrocompactMessages(msgs, 3)

	// Turn 1 tool result (read) should be cleared — it's before the cutoff.
	tr1 := result[2].(*types.ToolResultMessage)
	if tc, ok := tr1.Content[0].(*types.TextContent); ok {
		if tc.Text != "[Old tool result content cleared]" {
			t.Errorf("turn 1 read result should be cleared, got %q", tc.Text)
		}
	}

	// Turn 2 tool result (bash) should be cleared — it's before the cutoff.
	tr2 := result[4].(*types.ToolResultMessage)
	if tc, ok := tr2.Content[0].(*types.TextContent); ok {
		if tc.Text != "[Old tool result content cleared]" {
			t.Errorf("turn 2 bash result should be cleared, got %q", tc.Text)
		}
	}

	// Turn 3 tool result (edit) should NOT be cleared — edit is not clearable.
	tr3 := result[6].(*types.ToolResultMessage)
	if tc, ok := tr3.Content[0].(*types.TextContent); ok {
		if tc.Text != "edit result turn 3" {
			t.Errorf("turn 3 edit result should be preserved, got %q", tc.Text)
		}
	}

	// Turn 4 tool result (grep) should be preserved — it's within keepTurns.
	tr4 := result[8].(*types.ToolResultMessage)
	if tc, ok := tr4.Content[0].(*types.TextContent); ok {
		if tc.Text != "grep result turn 4" {
			t.Errorf("turn 4 grep result should be preserved, got %q", tc.Text)
		}
	}
}

func TestMicrocompactMessages_TooFewTurns(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{Role: "user"},
		&types.AssistantMessage{Role: "assistant", Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "hi"},
		}},
	}
	result := MicrocompactMessages(msgs, 3)
	// Should return unchanged — not enough turns.
	if len(result) != len(msgs) {
		t.Errorf("expected same length, got %d vs %d", len(result), len(msgs))
	}
}

func TestTruncateToolResult(t *testing.T) {
	t.Run("small result unchanged", func(t *testing.T) {
		content := []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "short"},
		}
		result := truncateToolResult(content)
		if tc := result[0].(*types.TextContent); tc.Text != "short" {
			t.Errorf("expected unchanged, got %q", tc.Text)
		}
	})

	t.Run("large result truncated", func(t *testing.T) {
		bigText := strings.Repeat("x", maxToolResultChars+10000)
		content := []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: bigText},
		}
		result := truncateToolResult(content)
		tc := result[0].(*types.TextContent)
		if len(tc.Text) >= len(bigText) {
			t.Errorf("expected truncation, got len=%d (original=%d)", len(tc.Text), len(bigText))
		}
		if !strings.Contains(tc.Text, "[truncated:") {
			t.Error("expected truncation marker")
		}
		// Should start with the head and end with the tail.
		if !strings.HasPrefix(tc.Text, "xxx") {
			t.Error("expected head preserved")
		}
		if !strings.HasSuffix(tc.Text, "xxx") {
			t.Error("expected tail preserved")
		}
	})

	t.Run("non-text blocks preserved", func(t *testing.T) {
		content := []types.ContentBlock{
			&types.ImageContent{Type: "image", URL: "http://example.com/img.png"},
			&types.TextContent{Type: types.ContentTypeText, Text: strings.Repeat("y", maxToolResultChars+1000)},
		}
		result := truncateToolResult(content)
		if len(result) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(result))
		}
		// Image should pass through.
		if _, ok := result[0].(*types.ImageContent); !ok {
			t.Error("first block should still be ImageContent")
		}
		// Text should be truncated.
		tc := result[1].(*types.TextContent)
		if !strings.Contains(tc.Text, "[truncated:") {
			t.Error("text block should be truncated")
		}
	})
}
