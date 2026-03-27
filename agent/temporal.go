package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// TemporalInjector is a TransformContext middleware that prepends a
// "[Date: YYYY-MM-DD HH:MM]" user message when the model needs a temporal
// anchor. It fires in three situations:
//
//  1. First message of a brand-new conversation (no history yet).
//  2. Session gap: the last message in history was sent more than GapThreshold
//     ago (default 30 minutes) — the user has come back after a break.
//  3. After context compression: the summary message produced by
//     ContextCompressor already carries a timestamp header, so the next turn
//     looks like a fresh start and condition 1 fires automatically.
type TemporalInjector struct {
	// GapThreshold is how long of a silence triggers a re-injection (default 30m).
	GapThreshold time.Duration
}

// NewTemporalInjector creates a TemporalInjector with the given gap threshold.
// Pass 0 to use the default of 30 minutes.
func NewTemporalInjector(gapThreshold time.Duration) *TemporalInjector {
	if gapThreshold <= 0 {
		gapThreshold = 30 * time.Minute
	}
	return &TemporalInjector{GapThreshold: gapThreshold}
}

// TransformContext implements the AgentLoopConfig.TransformContext hook.
// It prepends a lightweight temporal context message when warranted, then
// returns the (possibly extended) message slice unchanged otherwise.
func (t *TemporalInjector) TransformContext(_ context.Context, msgs []types.Message) ([]types.Message, error) {
	if !t.shouldInject(msgs) {
		return msgs, nil
	}

	anchor := &types.UserMessage{
		Role: "user",
		Content: []types.ContentBlock{
			&types.TextContent{
				Type: types.ContentTypeText,
				Text: fmt.Sprintf("[Temporal context: current date and time is %s]",
					time.Now().Format("2006-01-02 15:04 MST")),
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	out := make([]types.Message, 0, len(msgs)+1)
	out = append(out, anchor)
	out = append(out, msgs...)
	return out, nil
}

// shouldInject returns true when a temporal anchor should be injected.
func (t *TemporalInjector) shouldInject(msgs []types.Message) bool {
	if len(msgs) == 0 {
		return false // no messages at all — nothing to anchor
	}

	// Find the timestamp of the most recent message.
	lastTs := lastMessageTimestamp(msgs)
	if lastTs == 0 {
		// No timestamps available → treat as first message.
		return true
	}

	last := time.UnixMilli(lastTs)
	gap := time.Since(last)

	// First message: history only contains the user prompt just appended (gap < 1s).
	// We detect this by checking whether there are any *assistant* messages yet.
	if !hasAssistantMessage(msgs) {
		return true
	}

	// Session gap: user came back after a long pause.
	return gap >= t.GapThreshold
}

// lastMessageTimestamp returns the UnixMilli timestamp of the last message
// that carries one, or 0 if none do.
func lastMessageTimestamp(msgs []types.Message) int64 {
	for i := len(msgs) - 1; i >= 0; i-- {
		switch m := msgs[i].(type) {
		case *types.UserMessage:
			if m.Timestamp != 0 {
				return m.Timestamp
			}
		case *types.AssistantMessage:
			if m.Timestamp != 0 {
				return m.Timestamp
			}
		case *types.ToolResultMessage:
			if m.Timestamp != 0 {
				return m.Timestamp
			}
		}
	}
	return 0
}

// hasAssistantMessage reports whether msgs contains at least one assistant reply.
func hasAssistantMessage(msgs []types.Message) bool {
	for _, m := range msgs {
		if m.MessageRole() == "assistant" {
			return true
		}
	}
	return false
}
