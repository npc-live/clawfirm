package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/stream"
	"github.com/ai-gateway/clawfirm/types"
)

const (
	defaultBaseURL       = "https://api.anthropic.com"
	anthropicVersion     = "2023-06-01"
	defaultMaxTokens     = 8192
)

// Provider implements provider.LLMProvider for Anthropic's API.
type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// New creates an Anthropic Provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// NewWithBaseURL creates an Anthropic Provider with a custom base URL (for testing).
func NewWithBaseURL(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// ID returns "anthropic".
func (p *Provider) ID() string { return "anthropic" }

// Models returns the built-in Anthropic model list.
func (p *Provider) Models() []types.Model { return BuiltinModels() }

// --- request/response types ---

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []map[string]any
}

type anthropicTool struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"input_schema"`
	CacheControl *cacheControl  `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    any                `json:"system,omitempty"` // string or []map[string]any (structured with cache_control)
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream"`
	Thinking  *anthropicThinking `json:"thinking,omitempty"`
}

type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// --- SSE event types ---

type contentBlockStartEvent struct {
	Index        int `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
		Text string `json:"text"`
	} `json:"content_block"`
}

type contentBlockDeltaEvent struct {
	Index int `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		Thinking    string `json:"thinking"`
		Signature   string `json:"signature"`
	} `json:"delta"`
}

type messageDeltaEvent struct {
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type messageStartEvent struct {
	Message struct {
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// Stream sends a streaming request to Anthropic and returns an event channel.
func (p *Provider) Stream(ctx context.Context, req provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	apiKey := req.Options.APIKey
	if apiKey == "" {
		apiKey = p.apiKey
	}

	// Build messages
	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("anthropic: convert messages: %w", err)
	}

	// Build tools
	var tools []anthropicTool
	for _, t := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	// Mark the last tool with cache_control so the entire tools array is cached.
	if len(tools) > 0 {
		tools[len(tools)-1].CacheControl = &cacheControl{Type: "ephemeral"}
	}

	maxTokens := defaultMaxTokens
	if req.Options.MaxTokens != nil {
		maxTokens = *req.Options.MaxTokens
	} else if req.Model.MaxTokens > 0 {
		maxTokens = req.Model.MaxTokens
	}

	// System prompt: use structured block with cache_control for prompt caching.
	var system any
	if req.SystemPrompt != "" {
		system = []map[string]any{
			{
				"type":          "text",
				"text":          req.SystemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		}
	}

	// Inject message cache breakpoints for conversation history caching.
	injectMessageCacheBreakpoints(msgs)

	body := anthropicRequest{
		Model:     req.Model.ID,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  msgs,
		Tools:     tools,
		Stream:    true,
	}

	// Add thinking config if requested
	if req.Options.ThinkingLevel != "" && req.Options.ThinkingLevel != types.ThinkingLevelOff {
		budget := thinkingBudget(req.Options.ThinkingLevel, maxTokens)
		body.Thinking = &anthropicThinking{Type: "enabled", BudgetTokens: budget}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	baseURL := p.baseURL
	if req.Model.BaseURL != "" {
		baseURL = req.Model.BaseURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}
	// Native Anthropic keys start with "sk-ant-"; other tokens (e.g. sk-ss-v1- relay
	// keys) use Bearer auth as the Anthropic SDK does with ANTHROPIC_AUTH_TOKEN.
	if strings.HasPrefix(apiKey, "sk-ant-") {
		httpReq.Header.Set("x-api-key", apiKey)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")
	for k, v := range req.Options.Headers {
		httpReq.Header.Set(k, v)
	}

	log.Printf("anthropic: POST %s model=%s messages=%d", baseURL+"/v1/messages", req.Model.ID, len(msgs))
	resp, err := p.client.Do(httpReq)
	if err != nil {
		log.Printf("anthropic: http error: %v", err)
		return nil, fmt.Errorf("anthropic: http request: %w", err)
	}
	log.Printf("anthropic: HTTP %d", resp.StatusCode)

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Provider:   "anthropic",
			RetryAfter: provider.ParseRetryAfter(resp),
		}
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
	log.Printf("[anthropic-stream] starting stream read (model=%s)", modelID)

	sseEvents := stream.ParseSSEStream(ctx, body)

	// Accumulated state
	partial := &types.AssistantMessage{
		Role:     "assistant",
		Provider: "anthropic",
		Model:    modelID,
	}

	// Track tool call accumulation by content index
	type toolAccum struct {
		id   string
		name string
		args strings.Builder
	}
	toolAccums := map[int]*toolAccum{}
	// Track text accumulation by content index
	textAccums := map[int]*strings.Builder{}
	// Track content block types by index
	blockTypes := map[int]string{}

	var inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int

	emit := func(ev types.AssistantMessageEvent) {
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
	}

	emit(types.AssistantMessageEvent{Type: types.StreamEventStart})

	eventCount := 0
	for sseEv := range sseEvents {
		eventCount++
		if sseEv.Event == "" && sseEv.Data == "" {
			continue
		}
		switch sseEv.Event {
		case "message_start":
			log.Printf("[anthropic-stream] event: message_start")
			var ms messageStartEvent
			if err := json.Unmarshal([]byte(sseEv.Data), &ms); err == nil {
				inputTokens = ms.Message.Usage.InputTokens
				outputTokens = ms.Message.Usage.OutputTokens
				cacheReadTokens = ms.Message.Usage.CacheReadInputTokens
				cacheWriteTokens = ms.Message.Usage.CacheCreationInputTokens
			}

		case "content_block_start":
			var cbs contentBlockStartEvent
			if err := json.Unmarshal([]byte(sseEv.Data), &cbs); err != nil {
				continue
			}
			blockTypes[cbs.Index] = cbs.ContentBlock.Type
			switch cbs.ContentBlock.Type {
			case "text":
				textAccums[cbs.Index] = &strings.Builder{}
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventTextStart,
					ContentIndex: cbs.Index,
				})
			case "thinking":
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventThinkingStart,
					ContentIndex: cbs.Index,
				})
			case "tool_use":
				toolAccums[cbs.Index] = &toolAccum{
					id:   cbs.ContentBlock.ID,
					name: cbs.ContentBlock.Name,
				}
				tc := &types.ToolCall{
					Type: types.ContentTypeToolCall,
					ID:   cbs.ContentBlock.ID,
					Name: cbs.ContentBlock.Name,
				}
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventToolCallStart,
					ContentIndex: cbs.Index,
					ToolCall:     tc,
				})
			}

		case "content_block_delta":
			var cbd contentBlockDeltaEvent
			if err := json.Unmarshal([]byte(sseEv.Data), &cbd); err != nil {
				continue
			}
			btype := blockTypes[cbd.Index]
			switch btype {
			case "text":
				if acc, ok := textAccums[cbd.Index]; ok {
					acc.WriteString(cbd.Delta.Text)
				}
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventTextDelta,
					ContentIndex: cbd.Index,
					Delta:        cbd.Delta.Text,
				})
			case "thinking":
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventThinkingDelta,
					ContentIndex: cbd.Index,
					Delta:        cbd.Delta.Thinking,
				})
			case "tool_use":
				if acc, ok := toolAccums[cbd.Index]; ok {
					acc.args.WriteString(cbd.Delta.PartialJSON)
				}
			}

		case "content_block_stop":
			var cbs struct {
				Index int `json:"index"`
			}
			if err := json.Unmarshal([]byte(sseEv.Data), &cbs); err != nil {
				continue
			}
			btype := blockTypes[cbs.Index]
			switch btype {
			case "text":
				// Accumulate full text into partial.Content for history.
				if acc, ok := textAccums[cbs.Index]; ok {
					partial.Content = append(partial.Content, &types.TextContent{
						Type: types.ContentTypeText,
						Text: acc.String(),
					})
				}
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventTextEnd,
					ContentIndex: cbs.Index,
				})
			case "thinking":
				emit(types.AssistantMessageEvent{
					Type:         types.StreamEventThinkingEnd,
					ContentIndex: cbs.Index,
				})
			case "tool_use":
				if acc, ok := toolAccums[cbs.Index]; ok {
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
						ContentIndex: cbs.Index,
						ToolCall:     tc,
					})
				}
			}

		case "message_delta":
			var md messageDeltaEvent
			if err := json.Unmarshal([]byte(sseEv.Data), &md); err != nil {
				continue
			}
			outputTokens += md.Usage.OutputTokens
			partial.StopReason = mapStopReason(md.Delta.StopReason)
			log.Printf("[anthropic-stream] event: message_delta — stop_reason=%q", md.Delta.StopReason)

		case "message_stop":
			log.Printf("[anthropic-stream] event: message_stop — emitting Done (content=%d blocks)", len(partial.Content))
			partial.Usage = types.Usage{
				Input:      inputTokens,
				Output:     outputTokens,
				CacheRead:  cacheReadTokens,
				CacheWrite: cacheWriteTokens,
				Total:      inputTokens + outputTokens,
			}
			partial.Timestamp = time.Now().UnixMilli()
			finalMsg := *partial
			emit(types.AssistantMessageEvent{
				Type:    types.StreamEventDone,
				Message: &finalMsg,
				Reason:  finalMsg.StopReason,
			})
			return

		case "error":
			var errBody struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			_ = json.Unmarshal([]byte(sseEv.Data), &errBody)
			errMsg := errBody.Error.Message
			if errMsg == "" {
				errMsg = sseEv.Data
			}
			log.Printf("[anthropic-stream] event: error — %s", errMsg)
			errAMsg := &types.AssistantMessage{
				Role:         "assistant",
				Provider:     "anthropic",
				Model:        modelID,
				StopReason:   types.StopReasonError,
				ErrorMessage: errMsg,
				Timestamp:    time.Now().UnixMilli(),
			}
			emit(types.AssistantMessageEvent{
				Type:  types.StreamEventError,
				Error: errAMsg,
			})
			return
		default:
			log.Printf("[anthropic-stream] unknown event type: %q", sseEv.Event)
		}
	}
	// SSE channel closed without message_stop — stream was interrupted
	log.Printf("[anthropic-stream] WARNING: SSE channel closed without message_stop (events=%d, content=%d blocks, stopReason=%q)",
		eventCount, len(partial.Content), partial.StopReason)

	// Emit whatever we have so the loop doesn't hang forever
	if len(partial.Content) > 0 || partial.StopReason != "" {
		partial.Usage = types.Usage{
			Input:  inputTokens,
			Output: outputTokens,
			Total:  inputTokens + outputTokens,
		}
		partial.Timestamp = time.Now().UnixMilli()
		if partial.StopReason == "" {
			partial.StopReason = types.StopReasonError
			partial.ErrorMessage = "stream interrupted: connection closed without message_stop"
		}
		finalMsg := *partial
		emit(types.AssistantMessageEvent{
			Type:    types.StreamEventDone,
			Message: &finalMsg,
			Reason:  finalMsg.StopReason,
		})
	}
}

// mapStopReason maps Anthropic stop reasons to the canonical StopReason type.
func mapStopReason(reason string) types.StopReason {
	switch reason {
	case "end_turn":
		return types.StopReasonStop
	case "max_tokens":
		return types.StopReasonLength
	case "tool_use":
		return types.StopReasonToolUse
	default:
		return types.StopReasonStop
	}
}

// thinkingBudget returns an appropriate token budget for the given thinking level.
func thinkingBudget(level types.ThinkingLevel, maxTokens int) int {
	fraction := 0.5
	switch level {
	case types.ThinkingLevelMinimal:
		fraction = 0.1
	case types.ThinkingLevelLow:
		fraction = 0.2
	case types.ThinkingLevelMedium:
		fraction = 0.4
	case types.ThinkingLevelHigh:
		fraction = 0.6
	case types.ThinkingLevelXHigh:
		fraction = 0.8
	}
	budget := int(float64(maxTokens) * fraction)
	if budget < 1024 {
		budget = 1024
	}
	return budget
}

// convertMessages converts types.Message slice to Anthropic API format.
func convertMessages(msgs []types.Message) ([]anthropicMessage, error) {
	var out []anthropicMessage
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			content, err := convertContentBlocks(msg.Content)
			if err != nil {
				return nil, err
			}
			out = append(out, anthropicMessage{Role: "user", Content: content})
		case *types.AssistantMessage:
			content, err := convertContentBlocks(msg.Content)
			if err != nil {
				return nil, err
			}
			out = append(out, anthropicMessage{Role: "assistant", Content: content})
		case *types.ToolResultMessage:
			result := map[string]any{
				"type":        "tool_result",
				"tool_use_id": msg.ToolCallID,
				"is_error":    msg.IsError,
			}
			if len(msg.Content) > 0 {
				if tc, ok := msg.Content[0].(*types.TextContent); ok {
					result["content"] = tc.Text
				}
			}
			out = append(out, anthropicMessage{Role: "user", Content: []map[string]any{result}})
		}
	}
	return out, nil
}

// convertContentBlocks converts ContentBlock slice to Anthropic content block format.
func convertContentBlocks(blocks []types.ContentBlock) ([]map[string]any, error) {
	var out []map[string]any
	for _, b := range blocks {
		var item map[string]any
		switch block := b.(type) {
		case *types.TextContent:
			item = map[string]any{"type": "text", "text": block.Text}
		case *types.ImageContent:
			if block.URL != "" {
				item = map[string]any{
					"type": "image",
					"source": map[string]any{
						"type": "url",
						"url":  block.URL,
					},
				}
			} else {
				item = map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": block.MimeType,
						"data":       block.Data,
					},
				}
			}
		case *types.ToolCall:
			item = map[string]any{
				"type":  "tool_use",
				"id":    block.ID,
				"name":  block.Name,
				"input": block.Arguments,
			}
		case *types.ThinkingContent:
			if block.ThinkingSignature != "" {
				// Keep the signature for validation but omit the thinking text
				// to save tokens. Anthropic requires the signature to be present
				// in history but the full thinking text is not needed.
				item = map[string]any{
					"type":      "thinking",
					"thinking":  "",
					"signature": block.ThinkingSignature,
				}
			} else {
				// No signature — skip entirely (e.g. non-Anthropic thinking blocks).
				continue
			}
		default:
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

// injectMessageCacheBreakpoints adds cache_control to the second-to-last user
// message so that the entire prefix up to that point is cached by Anthropic.
// This means on the next turn, only the final user message + new tool results
// are "new" tokens — everything before is a cache hit.
func injectMessageCacheBreakpoints(msgs []anthropicMessage) {
	var userIndices []int
	for i, m := range msgs {
		if m.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}
	if len(userIndices) < 2 {
		return // too short to benefit from caching
	}
	targetIdx := userIndices[len(userIndices)-2]
	addCacheControlToLastBlock(&msgs[targetIdx])
}

// addCacheControlToLastBlock stamps cache_control on the last content block of
// an anthropicMessage. The Content field is `any` — it may be a string (simple
// user message) or []map[string]any (structured content blocks).
func addCacheControlToLastBlock(msg *anthropicMessage) {
	cc := map[string]string{"type": "ephemeral"}
	switch content := msg.Content.(type) {
	case []map[string]any:
		if len(content) > 0 {
			content[len(content)-1]["cache_control"] = cc
		}
	case string:
		// Convert from plain string to structured block so we can attach cache_control.
		msg.Content = []map[string]any{
			{
				"type":          "text",
				"text":          content,
				"cache_control": cc,
			},
		}
	}
}
