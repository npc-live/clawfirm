package message

import (
	"github.com/ai-gateway/pi-go/types"
)

// PruneMessages trims the message history to fit within maxTokens.
// The most recent messages are retained; older ones are dropped first.
// At least the last user message is always preserved.
func PruneMessages(messages []types.Message, maxTokens int) []types.Message {
	if len(messages) == 0 {
		return messages
	}

	// If within budget already, return as-is
	if EstimateTokens(messages) <= maxTokens {
		return messages
	}

	// Find the last user message index so we can guarantee it is kept
	lastUserIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].MessageRole() == "user" {
			lastUserIdx = i
			break
		}
	}

	// Start from the most recent messages and work backwards until we fit
	// Always include messages from lastUserIdx to end
	result := make([]types.Message, 0, len(messages))

	// Try keeping a growing suffix of messages that fits in the budget
	for start := len(messages) - 1; start >= 0; start-- {
		candidate := messages[start:]
		if lastUserIdx >= 0 && start > lastUserIdx {
			// We must include the last user message
			continue
		}
		if EstimateTokens(candidate) <= maxTokens {
			result = candidate
			break
		}
	}

	// If nothing fit (edge case: even the last user message alone exceeds budget),
	// keep just the last user message.
	if len(result) == 0 && lastUserIdx >= 0 {
		result = messages[lastUserIdx:]
	}

	return result
}
