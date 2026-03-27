package openai

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

const (
	defaultBaseURL   = "https://api.openai.com"
	defaultMaxTokens = 4096
)

// Provider implements provider.LLMProvider for OpenAI's API.
type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// New creates an OpenAI Provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// NewWithBaseURL creates an OpenAI Provider with a custom base URL (for testing).
func NewWithBaseURL(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// ID returns "openai".
func (p *Provider) ID() string { return "openai" }

// Models returns the built-in OpenAI model list.
func (p *Provider) Models() []types.Model { return BuiltinModels() }

// --- request/response types ---

type openAIMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content"` // string or []map[string]any
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

// OpenAI SSE chunk
type openAIChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string           `json:"role"`
			Content   *string          `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Stream sends a streaming request to OpenAI and returns an event channel.
func (p *Provider) Stream(ctx context.Context, req provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	apiKey := req.Options.APIKey
	if apiKey == "" {
		apiKey = p.apiKey
	}

	msgs := convertMessages(req.Messages, req.SystemPrompt)

	var tools []openAITool
	for _, t := range req.Tools {
		ot := openAITool{Type: "function"}
		ot.Function.Name = t.Name
		ot.Function.Description = t.Description
		ot.Function.Parameters = t.Parameters
		tools = append(tools, ot)
	}

	maxTokens := defaultMaxTokens
	if req.Options.MaxTokens != nil {
		maxTokens = *req.Options.MaxTokens
	} else if req.Model.MaxTokens > 0 {
		maxTokens = req.Model.MaxTokens
	}

	body := openAIRequest{
		Model:       req.Model.ID,
		Messages:    msgs,
		Tools:       tools,
		MaxTokens:   maxTokens,
		Temperature: req.Options.Temperature,
		Stream:      true,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	baseURL := p.baseURL
	if req.Model.BaseURL != "" {
		baseURL = req.Model.BaseURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Options.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: http request: %w", err)
	}

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, string(b))
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
		Provider: "openai",
		Model:    modelID,
	}

	// Accumulate text and tool call data
	var textBuf strings.Builder
	// tool call accumulation: indexed by tool call index
	type toolAccum struct {
		id   string
		name string
		args strings.Builder
	}
	toolAccums := map[int]*toolAccum{}

	var textStarted bool
	var inputTokens, outputTokens int

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		if data == "" {
			continue
		}

		var chunk openAIChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			// Text content
			if delta.Content != nil {
				if !textStarted {
					emit(types.AssistantMessageEvent{Type: types.StreamEventTextStart, ContentIndex: 0})
					textStarted = true
				}
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventTextDelta,
					ContentIndex: 0,
					Delta:        *delta.Content,
				})
				textBuf.WriteString(*delta.Content)
			}

			// Tool calls
			for _, tc := range delta.ToolCalls {
				idx := tc.Index
				if _, ok := toolAccums[idx]; !ok {
					toolAccums[idx] = &toolAccum{}
				}
				acc := toolAccums[idx]
				if tc.ID != "" {
					acc.id = tc.ID
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				acc.args.WriteString(tc.Function.Arguments)
			}

			// Finish
			if choice.FinishReason != nil {
				stopReason := mapStopReason(*choice.FinishReason)
				partial.StopReason = stopReason

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

				for idx, acc := range toolAccums {
					var args map[string]any
					_ = json.Unmarshal([]byte(acc.args.String()), &args)
					if args == nil {
						args = map[string]any{}
					}
					tc := &types.ToolCall{
						Type:      types.ContentTypeToolCall,
						ID:        acc.id,
						Name:      acc.name,
						Arguments: args,
					}
					partial.Content = append(partial.Content, tc)
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventToolCallEnd,
						ContentIndex: idx,
						ToolCall:     tc,
					})
				}
			}
		}
	}

	partial.Usage = types.Usage{
		Input:  inputTokens,
		Output: outputTokens,
		Total:  inputTokens + outputTokens,
	}
	partial.Timestamp = time.Now().UnixMilli()
	finalMsg := *partial
	emit(types.AssistantMessageEvent{
		Type:    types.StreamEventDone,
		Message: &finalMsg,
		Reason:  finalMsg.StopReason,
	})
}

// mapStopReason maps OpenAI finish reasons to canonical StopReason.
func mapStopReason(reason string) types.StopReason {
	switch reason {
	case "stop":
		return types.StopReasonStop
	case "length":
		return types.StopReasonLength
	case "tool_calls":
		return types.StopReasonToolUse
	default:
		return types.StopReasonStop
	}
}

// convertMessages converts types.Message slice to OpenAI API format.
func convertMessages(msgs []types.Message, systemPrompt string) []openAIMessage {
	var out []openAIMessage
	if systemPrompt != "" {
		out = append(out, openAIMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			// Simple text only for now; multimodal content handled similarly
			var textParts []string
			for _, c := range msg.Content {
				if tc, ok := c.(*types.TextContent); ok {
					textParts = append(textParts, tc.Text)
				}
			}
			content := strings.Join(textParts, "\n")
			out = append(out, openAIMessage{Role: "user", Content: content})
		case *types.AssistantMessage:
			oam := openAIMessage{Role: "assistant"}
			var textParts []string
			var toolCalls []openAIToolCall
			for _, c := range msg.Content {
				switch block := c.(type) {
				case *types.TextContent:
					textParts = append(textParts, block.Text)
				case *types.ToolCall:
					argsBytes, _ := json.Marshal(block.Arguments)
					tc := openAIToolCall{
						ID:   block.ID,
						Type: "function",
					}
					tc.Function.Name = block.Name
					tc.Function.Arguments = string(argsBytes)
					toolCalls = append(toolCalls, tc)
				}
			}
			if len(textParts) > 0 {
				oam.Content = strings.Join(textParts, "\n")
			}
			oam.ToolCalls = toolCalls
			out = append(out, oam)
		case *types.ToolResultMessage:
			var text string
			if len(msg.Content) > 0 {
				if tc, ok := msg.Content[0].(*types.TextContent); ok {
					text = tc.Text
				}
			}
			out = append(out, openAIMessage{
				Role:       "tool",
				Content:    text,
				ToolCallID: msg.ToolCallID,
			})
		}
	}
	return out
}
