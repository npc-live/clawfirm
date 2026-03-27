package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-gateway/pi-go/types"
)

// CompressorConfig controls when and how context compression is triggered.
type CompressorConfig struct {
	// Threshold is the fraction of ContextWindow at which compression fires (default 0.8).
	Threshold float64
	// ContextWindow is the model's total token capacity. When 0, compression is disabled.
	ContextWindow int
	// KeepLastN is the number of recent messages to preserve verbatim after compression (default 4).
	KeepLastN int
}

func (c CompressorConfig) threshold() float64 {
	if c.Threshold <= 0 || c.Threshold > 1 {
		return 0.8
	}
	return c.Threshold
}

func (c CompressorConfig) keepLastN() int {
	if c.KeepLastN <= 0 {
		return 4
	}
	return c.KeepLastN
}

// SummarizeMessagesFunc summarizes a list of messages to a single string.
type SummarizeMessagesFunc func(ctx context.Context, msgs []types.Message) (string, error)

// ContextCompressor implements a TransformContext hook that compresses message
// history when token usage approaches the configured threshold.
//
// Algorithm:
//  1. Estimate total tokens in the current message slice.
//  2. If tokens < threshold × ContextWindow → no-op.
//  3. Call summarizeFn on all messages except the last keepLastN.
//  4. Replace the summarized messages with a single UserMessage containing
//     a "[Context Summary]" block, then re-append the kept tail.
type ContextCompressor struct {
	cfg         CompressorConfig
	summarizeFn SummarizeMessagesFunc
}

// NewContextCompressor creates a ContextCompressor.
// summarizeFn must be non-nil; cfg.ContextWindow must be > 0 for compression to activate.
func NewContextCompressor(summarizeFn SummarizeMessagesFunc, cfg CompressorConfig) *ContextCompressor {
	return &ContextCompressor{cfg: cfg, summarizeFn: summarizeFn}
}

// TransformContext is the AgentLoopConfig.TransformContext hook.
// It returns the (possibly compressed) message slice.
func (c *ContextCompressor) TransformContext(ctx context.Context, msgs []types.Message) ([]types.Message, error) {
	if c.cfg.ContextWindow <= 0 || len(msgs) == 0 {
		return msgs, nil
	}

	used := estimateMessagesTokens(msgs)
	limit := int(float64(c.cfg.ContextWindow) * c.cfg.threshold())
	if used < limit {
		return msgs, nil
	}

	keepN := c.cfg.keepLastN()
	if keepN >= len(msgs) {
		// Nothing left to summarize.
		return msgs, nil
	}

	head := msgs[:len(msgs)-keepN]
	tail := msgs[len(msgs)-keepN:]

	summary, err := c.summarizeFn(ctx, head)
	if err != nil {
		// Compression failed; return original to avoid breaking the loop.
		return msgs, fmt.Errorf("compressor: summarize: %w", err)
	}

	summaryMsg := buildSummaryMessage(summary)
	compressed := make([]types.Message, 0, 1+len(tail))
	compressed = append(compressed, summaryMsg)
	compressed = append(compressed, tail...)
	return compressed, nil
}

// buildSummaryMessage wraps a summary string in a UserMessage with a clear label.
func buildSummaryMessage(summary string) *types.UserMessage {
	text := fmt.Sprintf("[Context Summary — %s]\n\n%s",
		time.Now().Format("2006-01-02 15:04"), summary)
	return &types.UserMessage{
		Role: "user",
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: text},
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

// estimateMessagesTokens estimates the total token count across all messages.
// Uses the same 1 token ≈ 4 chars heuristic as memory/chunk.go.
func estimateMessagesTokens(msgs []types.Message) int {
	total := 0
	for _, m := range msgs {
		total += estimateMessageTokens(m)
	}
	return total
}

func estimateMessageTokens(m types.Message) int {
	switch msg := m.(type) {
	case *types.UserMessage:
		return estimateBlockTokens(msg.Content)
	case *types.AssistantMessage:
		return msg.Usage.Input + msg.Usage.Output + estimateBlockTokens(msg.Content)
	case *types.ToolResultMessage:
		return estimateBlockTokens(msg.Content)
	}
	return 0
}

func estimateBlockTokens(blocks []types.ContentBlock) int {
	total := 0
	for _, b := range blocks {
		switch block := b.(type) {
		case *types.TextContent:
			total += (len(block.Text) + 3) / 4
		case *types.ThinkingContent:
			total += (len(block.Thinking) + 3) / 4
		default:
			total += 256 // image / audio / tool call placeholder
		}
	}
	return total
}

// BuildSummarizeContextPrompt formats msgs into a plain-text prompt for context compression.
func BuildSummarizeContextPrompt(msgs []types.Message) string {
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation history concisely.\n")
	sb.WriteString("Preserve key decisions, facts, code snippets, and unresolved issues.\n\n")
	sb.WriteString("---\n\n")
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			sb.WriteString("User: ")
			sb.WriteString(extractContentText(msg.Content))
			sb.WriteString("\n\n")
		case *types.AssistantMessage:
			sb.WriteString("Assistant: ")
			sb.WriteString(extractContentText(msg.Content))
			sb.WriteString("\n\n")
		case *types.ToolResultMessage:
			sb.WriteString("Tool [")
			sb.WriteString(msg.ToolName)
			sb.WriteString("]: ")
			sb.WriteString(extractContentText(msg.Content))
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

func extractContentText(blocks []types.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if t, ok := b.(*types.TextContent); ok {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " ")
}
