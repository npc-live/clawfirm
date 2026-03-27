package message

import (
	"testing"

	"github.com/ai-gateway/pi-go/types"
)

func TestFilterForLLMRemovesURLAudio(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "hello"},
				&types.AudioContent{Type: types.ContentTypeAudio, URL: "https://example.com/audio.mp3", MimeType: "audio/mp3"},
			},
		},
	}

	result := FilterForLLM(msgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	um, ok := result[0].(*types.UserMessage)
	if !ok {
		t.Fatalf("expected *UserMessage, got %T", result[0])
	}
	if len(um.Content) != 1 {
		t.Errorf("expected 1 content block (audio removed), got %d", len(um.Content))
	}
	if um.Content[0].ContentType() != types.ContentTypeText {
		t.Errorf("expected text content, got %q", um.Content[0].ContentType())
	}
}

func TestFilterForLLMKeepsInlineAudio(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.AudioContent{Type: types.ContentTypeAudio, Data: "base64data", MimeType: "audio/mp3"},
			},
		},
	}

	result := FilterForLLM(msgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	um := result[0].(*types.UserMessage)
	if len(um.Content) != 1 {
		t.Errorf("expected 1 content block, got %d", len(um.Content))
	}
	if um.Content[0].ContentType() != types.ContentTypeAudio {
		t.Errorf("expected audio content, got %q", um.Content[0].ContentType())
	}
}

func TestFilterForLLMSkipsEmptyUserMessage(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				// only a URL-only audio block → entire message is dropped
				&types.AudioContent{Type: types.ContentTypeAudio, URL: "https://x.com/a.mp3", MimeType: "audio/mp3"},
			},
		},
	}
	result := FilterForLLM(msgs)
	if len(result) != 0 {
		t.Errorf("expected 0 messages (all content filtered), got %d", len(result))
	}
}

func TestFilterForLLMPassesThroughAssistantMessages(t *testing.T) {
	msgs := []types.Message{
		&types.AssistantMessage{
			Role:    "assistant",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "hi"}},
		},
	}
	result := FilterForLLM(msgs)
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}

func TestEstimateTokensBasic(t *testing.T) {
	msgs := []types.Message{
		&types.UserMessage{
			Role:    "user",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "hello world"}},
		},
	}
	tokens := EstimateTokens(msgs)
	if tokens <= 0 {
		t.Errorf("expected positive token estimate, got %d", tokens)
	}
}

func TestEstimateTokensMultipleMessages(t *testing.T) {
	short := []types.Message{
		&types.UserMessage{
			Role:    "user",
			Content: []types.ContentBlock{&types.TextContent{Type: types.ContentTypeText, Text: "hi"}},
		},
	}
	long := []types.Message{
		&types.UserMessage{
			Role: "user",
			Content: []types.ContentBlock{
				&types.TextContent{Type: types.ContentTypeText, Text: "This is a much longer message with lots of tokens to estimate. " +
					"We expect this to return a higher token count than a short message."},
			},
		},
	}

	shortTokens := EstimateTokens(short)
	longTokens := EstimateTokens(long)
	if longTokens <= shortTokens {
		t.Errorf("expected long message to have more tokens (%d > %d)", longTokens, shortTokens)
	}
}
