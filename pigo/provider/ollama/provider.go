package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/types"
)

const defaultBaseURL = "http://localhost:11434"

// Provider implements provider.LLMProvider for locally-running Ollama.
type Provider struct {
	baseURL string
	client  *http.Client
}

// New creates an Ollama Provider connecting to the default local URL.
func New() *Provider {
	return &Provider{
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// NewWithBaseURL creates an Ollama Provider with a custom base URL.
func NewWithBaseURL(baseURL string) *Provider {
	return &Provider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// ID returns "ollama".
func (p *Provider) ID() string { return "ollama" }

// Models returns an empty list; use DiscoverModels for dynamic discovery.
func (p *Provider) Models() []types.Model { return nil }

// --- Ollama API types ---

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaChunk struct {
	Model     string        `json:"model"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
	DoneReason string       `json:"done_reason,omitempty"`
}

// Stream sends a streaming request to Ollama and returns an event channel.
func (p *Provider) Stream(ctx context.Context, req provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	msgs := convertMessages(req.Messages, req.SystemPrompt)

	body := ollamaRequest{
		Model:    req.Model.ID,
		Messages: msgs,
		Stream:   true,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	baseURL := p.baseURL
	if req.Model.BaseURL != "" {
		baseURL = req.Model.BaseURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: http request: %w", err)
	}

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, string(b))
	}

	ch := make(chan types.AssistantMessageEvent, 32)
	go func() {
		defer close(ch)
		p.readStream(ctx, resp.Body, ch, req.Model.ID)
	}()
	return ch, nil
}

func (p *Provider) readStream(ctx context.Context, body io.ReadCloser, ch chan<- types.AssistantMessageEvent, modelID string) {
	defer body.Close()

	emit := func(ev types.AssistantMessageEvent) {
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
	}

	emit(types.AssistantMessageEvent{Type: types.StreamEventStart})

	partial := &types.AssistantMessage{
		Role:     "assistant",
		Provider: "ollama",
		Model:    modelID,
	}

	var textBuf strings.Builder
	var textStarted bool

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk ollamaChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		// Each non-done chunk carries a delta in message.content
		if chunk.Message.Content != "" {
			if !textStarted {
				emit(types.AssistantMessageEvent{Type: types.StreamEventTextStart, ContentIndex: 0})
				textStarted = true
			}
			emit(types.AssistantMessageEvent{
				Type:         types.StreamEventTextDelta,
				ContentIndex: 0,
				Delta:        chunk.Message.Content,
			})
			textBuf.WriteString(chunk.Message.Content)
		}

		if chunk.Done {
			if textStarted {
				full := textBuf.String()
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventTextEnd,
					ContentIndex: 0,
					Content:      full,
				})
				partial.Content = append(partial.Content, &types.TextContent{
					Type: types.ContentTypeText,
					Text: full,
				})
			}
			partial.StopReason = types.StopReasonStop
			partial.Timestamp = time.Now().UnixMilli()
			finalMsg := *partial
			emit(types.AssistantMessageEvent{
				Type:    types.StreamEventDone,
				Message: &finalMsg,
				Reason:  types.StopReasonStop,
			})
			return
		}
	}

	// Stream ended without done=true
	if textStarted {
		full := textBuf.String()
		partial.Content = append(partial.Content, &types.TextContent{
			Type: types.ContentTypeText,
			Text: full,
		})
	}
	partial.StopReason = types.StopReasonStop
	partial.Timestamp = time.Now().UnixMilli()
	finalMsg := *partial
	emit(types.AssistantMessageEvent{
		Type:    types.StreamEventDone,
		Message: &finalMsg,
		Reason:  types.StopReasonStop,
	})
}

// convertMessages converts types.Message slice to Ollama's simple chat format.
func convertMessages(msgs []types.Message, systemPrompt string) []ollamaMessage {
	var out []ollamaMessage
	if systemPrompt != "" {
		out = append(out, ollamaMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			var parts []string
			for _, c := range msg.Content {
				if tc, ok := c.(*types.TextContent); ok {
					parts = append(parts, tc.Text)
				}
			}
			out = append(out, ollamaMessage{Role: "user", Content: strings.Join(parts, "\n")})
		case *types.AssistantMessage:
			var parts []string
			for _, c := range msg.Content {
				if tc, ok := c.(*types.TextContent); ok {
					parts = append(parts, tc.Text)
				}
			}
			out = append(out, ollamaMessage{Role: "assistant", Content: strings.Join(parts, "\n")})
		case *types.ToolResultMessage:
			var text string
			if len(msg.Content) > 0 {
				if tc, ok := msg.Content[0].(*types.TextContent); ok {
					text = tc.Text
				}
			}
			out = append(out, ollamaMessage{Role: "tool", Content: text})
		}
	}
	return out
}
