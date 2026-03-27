package message

import (
	"encoding/json"

	"github.com/ai-gateway/pi-go/types"
)

// FilterForLLM removes messages or content blocks that LLMs cannot process.
// Currently removes AudioContent blocks that lack transcription (URL-only audio).
func FilterForLLM(messages []types.Message) []types.Message {
	out := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		switch msg := m.(type) {
		case *types.UserMessage:
			filtered := filterContentBlocks(msg.Content)
			if len(filtered) == 0 {
				continue
			}
			out = append(out, &types.UserMessage{
				Role:      msg.Role,
				Content:   filtered,
				Timestamp: msg.Timestamp,
			})
		default:
			out = append(out, m)
		}
	}
	return out
}

// filterContentBlocks removes content blocks unsupported by the LLM.
func filterContentBlocks(blocks []types.ContentBlock) []types.ContentBlock {
	out := make([]types.ContentBlock, 0, len(blocks))
	for _, b := range blocks {
		switch block := b.(type) {
		case *types.AudioContent:
			// Skip audio without inline data (not yet transcribed)
			if block.Data == "" {
				continue
			}
			out = append(out, b)
		default:
			out = append(out, b)
		}
	}
	return out
}

// EstimateTokens estimates the total token count for a message slice.
// Uses the simple heuristic: total character count / 4.
func EstimateTokens(messages []types.Message) int {
	var total int
	for _, m := range messages {
		total += estimateMessageTokens(m)
	}
	return total
}

// estimateMessageTokens estimates tokens for a single message.
func estimateMessageTokens(m types.Message) int {
	// Serialize to JSON and count characters / 4
	data, err := json.Marshal(m)
	if err != nil {
		return 0
	}
	return len(data) / 4
}
