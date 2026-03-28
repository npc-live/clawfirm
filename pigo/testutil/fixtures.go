package testutil

import (
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// UserTextMessage creates a UserMessage containing a single text block.
func UserTextMessage(text string) types.UserMessage {
	return types.UserMessage{
		Role: "user",
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: text},
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

// UserImageMessage creates a UserMessage containing a text block and an image block.
func UserImageMessage(text string, imageData string, mimeType string) types.UserMessage {
	return types.UserMessage{
		Role: "user",
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: text},
			&types.ImageContent{Type: types.ContentTypeImage, Data: imageData, MimeType: mimeType},
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

// AssistantTextMessage creates an AssistantMessage containing a single text block.
func AssistantTextMessage(text string) types.AssistantMessage {
	return types.AssistantMessage{
		Role: "assistant",
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: text},
		},
		StopReason: types.StopReasonStop,
		Timestamp:  time.Now().UnixMilli(),
	}
}

// AssistantToolCallMessage creates an AssistantMessage containing a single tool call.
func AssistantToolCallMessage(toolName string, args map[string]any) types.AssistantMessage {
	return types.AssistantMessage{
		Role: "assistant",
		Content: []types.ContentBlock{
			&types.ToolCall{
				Type:      types.ContentTypeToolCall,
				ID:        "call_" + toolName,
				Name:      toolName,
				Arguments: args,
			},
		},
		StopReason: types.StopReasonToolUse,
		Timestamp:  time.Now().UnixMilli(),
	}
}

// ToolResultTextMessage creates a successful ToolResultMessage with text content.
func ToolResultTextMessage(callID, toolName, result string) types.ToolResultMessage {
	return types.ToolResultMessage{
		Role:       "tool",
		ToolCallID: callID,
		ToolName:   toolName,
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: result},
		},
		IsError:   false,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ToolResultErrorMessage creates a ToolResultMessage representing a tool error.
func ToolResultErrorMessage(callID, toolName, errMsg string) types.ToolResultMessage {
	return types.ToolResultMessage{
		Role:       "tool",
		ToolCallID: callID,
		ToolName:   toolName,
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: errMsg},
		},
		IsError:   true,
		Timestamp: time.Now().UnixMilli(),
	}
}
