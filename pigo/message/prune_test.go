package message

import (
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

func makeTextMsg(role, text string) types.Message {
	if role == "user" {
		return &types.UserMessage{
			Role:    "user",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
		}
	}
	return &types.AssistantMessage{
		Role:    "assistant",
		Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: text}},
	}
}

func TestPruneMessagesNoTrim(t *testing.T) {
	msgs := []types.Message{
		makeTextMsg("user", "hi"),
		makeTextMsg("assistant", "hello"),
	}
	result := PruneMessages(msgs, 100000)
	if len(result) != 2 {
		t.Errorf("expected 2 messages unchanged, got %d", len(result))
	}
}

func TestPruneMessagesKeepsLastUser(t *testing.T) {
	// Create messages that together exceed the budget
	// Each message is ~50 chars, budget is very small
	msgs := []types.Message{
		makeTextMsg("user", "first user message - very old"),
		makeTextMsg("assistant", "first assistant reply - very old"),
		makeTextMsg("user", "second user message - old"),
		makeTextMsg("assistant", "second assistant reply - old"),
		makeTextMsg("user", "last user message"),
	}

	// Tiny budget that forces pruning but must keep last user message
	result := PruneMessages(msgs, 1) // nearly zero budget
	if len(result) == 0 {
		t.Fatal("expected at least 1 message (last user)")
	}
	// Last message should be a user message
	last := result[len(result)-1]
	if last.MessageRole() != "user" {
		t.Errorf("last message role: got %q want user", last.MessageRole())
	}
	// The kept user message should be the last one
	if um, ok := last.(*types.UserMessage); ok {
		tc := um.Content[0].(*types.TextContent)
		if tc.Text != "last user message" {
			t.Errorf("last user message text: got %q want 'last user message'", tc.Text)
		}
	}
}

func TestPruneMessagesReturnsSameWhenFit(t *testing.T) {
	msgs := []types.Message{
		makeTextMsg("user", "hi"),
	}
	result := PruneMessages(msgs, 10000)
	if len(result) != len(msgs) {
		t.Errorf("expected %d messages, got %d", len(msgs), len(result))
	}
}

func TestPruneMessagesEmpty(t *testing.T) {
	result := PruneMessages(nil, 1000)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(result))
	}
}

func TestPruneMessagesTrimOlderMessages(t *testing.T) {
	// Build a long conversation
	msgs := make([]types.Message, 0, 20)
	for i := 0; i < 10; i++ {
		msgs = append(msgs,
			makeTextMsg("user", "user message number "+string(rune('0'+i))+" with some extra text to consume tokens"),
			makeTextMsg("assistant", "assistant reply to message "+string(rune('0'+i))+" with text to consume tokens"),
		)
	}
	// Last message is a user message
	msgs = append(msgs, makeTextMsg("user", "final question"))

	// Budget: enough for ~5 messages but not all 21
	tokens := EstimateTokens(msgs)
	budget := tokens / 3

	result := PruneMessages(msgs, budget)
	if len(result) >= len(msgs) {
		t.Errorf("expected pruning to reduce messages; got %d (same as original %d)", len(result), len(msgs))
	}
	// Last message must still be "final question"
	last := result[len(result)-1].(*types.UserMessage)
	if last.Content[0].(*types.TextContent).Text != "final question" {
		t.Errorf("last message not preserved after pruning")
	}
}
