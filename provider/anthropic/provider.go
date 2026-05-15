package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/types"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultMaxTokens = 8192
)

// Provider implements provider.LLMProvider using the official Anthropic Go SDK.
type Provider struct {
	apiKey  string
	baseURL string
	client  anthropicsdk.Client
}

func New(apiKey string) *Provider {
	return NewWithBaseURL(apiKey, defaultBaseURL)
}

func NewWithBaseURL(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  buildClient(apiKey, baseURL),
	}
}

func buildClient(apiKey, baseURL string) anthropicsdk.Client {
	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
		option.WithHTTPClient(provider.NewStreamingHTTPClient()),
	}
	if strings.HasPrefix(apiKey, "sk-ant-") {
		opts = append(opts, option.WithAPIKey(apiKey))
	} else {
		opts = append(opts, option.WithAuthToken(apiKey))
	}
	return anthropicsdk.NewClient(opts...)
}

func (p *Provider) ID() string            { return "anthropic" }
func (p *Provider) Models() []types.Model { return BuiltinModels() }

// Stream sends a streaming request to Anthropic and returns an event channel.
func (p *Provider) Stream(ctx context.Context, req provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	var reqOpts []option.RequestOption
	if req.Options.APIKey != "" {
		key := req.Options.APIKey
		if strings.HasPrefix(key, "sk-ant-") {
			reqOpts = append(reqOpts, option.WithAPIKey(key))
		} else {
			reqOpts = append(reqOpts, option.WithAuthToken(key))
		}
	}
	if req.Model.BaseURL != "" {
		reqOpts = append(reqOpts, option.WithBaseURL(req.Model.BaseURL))
	}
	for k, v := range req.Options.Headers {
		reqOpts = append(reqOpts, option.WithHeader(k, v))
	}

	maxTokens := defaultMaxTokens
	if req.Options.MaxTokens != nil {
		maxTokens = *req.Options.MaxTokens
	} else if req.Model.MaxTokens > 0 {
		maxTokens = req.Model.MaxTokens
	}

	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("anthropic: convert messages: %w", err)
	}
	injectCacheBreakpoints(msgs)

	params := anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(req.Model.ID),
		MaxTokens: int64(maxTokens),
		Messages:  msgs,
		Tools:     buildTools(req.Tools),
	}
	if req.SystemPrompt != "" {
		params.System = []anthropicsdk.TextBlockParam{{
			Text:         req.SystemPrompt,
			CacheControl: anthropicsdk.NewCacheControlEphemeralParam(),
		}}
	}
	if req.Options.ThinkingLevel != "" && req.Options.ThinkingLevel != types.ThinkingLevelOff {
		budget := thinkingBudget(req.Options.ThinkingLevel, maxTokens)
		params.Thinking = anthropicsdk.ThinkingConfigParamOfEnabled(int64(budget))
	}

	baseURL := p.baseURL
	if req.Model.BaseURL != "" {
		baseURL = req.Model.BaseURL
	}
	log.Printf("anthropic: POST %s model=%s messages=%d", baseURL+"/v1/messages", req.Model.ID, len(msgs))

	stream := p.client.Messages.NewStreaming(ctx, params, reqOpts...)

	ch := make(chan types.AssistantMessageEvent, 32)
	go func() {
		defer close(ch)
		defer stream.Close()

		streamStart := time.Now()
		log.Printf("[anthropic-stream] starting stream read (model=%s) at %v", req.Model.ID, streamStart.Format(time.RFC3339Nano))

		partial := &types.AssistantMessage{
			Role:     "assistant",
			Provider: "anthropic",
			Model:    req.Model.ID,
		}

		type toolAccum struct {
			id   string
			name string
			args strings.Builder
		}
		toolAccums := map[int]*toolAccum{}
		textAccums := map[int]*strings.Builder{}
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
		lastEventTime := streamStart

		for stream.Next() {
			ev := stream.Current()
			eventCount++
			elapsed := time.Since(streamStart)
			gap := time.Since(lastEventTime)
			lastEventTime = time.Now()

			if gap > 5*time.Second {
				log.Printf("[anthropic-stream] *** long gap before event #%d: %v (elapsed=%v) ***",
					eventCount, gap.Round(time.Millisecond), elapsed.Round(time.Millisecond))
			} else {
				log.Printf("[anthropic-stream] event #%d type=%q elapsed=%v gap=%v",
					eventCount, ev.Type, elapsed.Round(time.Millisecond), gap.Round(time.Millisecond))
			}

			switch variant := ev.AsAny().(type) {
			case anthropicsdk.MessageStartEvent:
				inputTokens = int(variant.Message.Usage.InputTokens)
				cacheReadTokens = int(variant.Message.Usage.CacheReadInputTokens)
				cacheWriteTokens = int(variant.Message.Usage.CacheCreationInputTokens)
				log.Printf("[anthropic-stream] event: message_start")

			case anthropicsdk.ContentBlockStartEvent:
				idx := int(variant.Index)
				cb := variant.ContentBlock
				blockTypes[idx] = cb.Type
				switch cb.Type {
				case "text":
					textAccums[idx] = &strings.Builder{}
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventTextStart,
						ContentIndex: idx,
					})
				case "thinking":
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventThinkingStart,
						ContentIndex: idx,
					})
				case "tool_use":
					log.Printf("[anthropic-stream] tool_use block #%d: id=%q name=%q", idx, cb.ID, cb.Name)
					toolAccums[idx] = &toolAccum{id: cb.ID, name: cb.Name}
					tc := &types.ToolCall{
						Type: types.ContentTypeToolCall,
						ID:   cb.ID,
						Name: cb.Name,
					}
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventToolCallStart,
						ContentIndex: idx,
						ToolCall:     tc,
					})
				}

			case anthropicsdk.ContentBlockDeltaEvent:
				idx := int(variant.Index)
				delta := variant.Delta
				switch blockTypes[idx] {
				case "text":
					text := delta.Text
					if acc, ok := textAccums[idx]; ok {
						acc.WriteString(text)
					}
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventTextDelta,
						ContentIndex: idx,
						Delta:        text,
					})
				case "thinking":
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventThinkingDelta,
						ContentIndex: idx,
						Delta:        delta.Thinking,
					})
				case "tool_use":
					if acc, ok := toolAccums[idx]; ok {
						pj := delta.PartialJSON
						acc.args.WriteString(pj)
						log.Printf("[anthropic-stream] tool_use #%d (%s) partial_json delta=%d total=%d",
							idx, acc.name, len(pj), acc.args.Len())
					}
				}

			case anthropicsdk.ContentBlockStopEvent:
				idx := int(variant.Index)
				switch blockTypes[idx] {
				case "text":
					if acc, ok := textAccums[idx]; ok {
						partial.Content = append(partial.Content, &types.TextContent{
							Type: types.ContentTypeText,
							Text: acc.String(),
						})
					}
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventTextEnd,
						ContentIndex: idx,
					})
				case "thinking":
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventThinkingEnd,
						ContentIndex: idx,
					})
				case "tool_use":
					if acc, ok := toolAccums[idx]; ok {
						var args map[string]any
						if err := json.Unmarshal([]byte(acc.args.String()), &args); err != nil {
							log.Printf("[anthropic-stream] malformed tool args for %s: %v", acc.name, err)
						}
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

			case anthropicsdk.MessageDeltaEvent:
				outputTokens += int(variant.Usage.OutputTokens)
				partial.StopReason = mapStopReason(string(variant.Delta.StopReason))
				log.Printf("[anthropic-stream] event: message_delta — stop_reason=%q elapsed=%v out_tokens=%d",
					variant.Delta.StopReason, time.Since(streamStart).Round(time.Millisecond), outputTokens)

			case anthropicsdk.MessageStopEvent:
				log.Printf("[anthropic-stream] event: message_stop — emitting Done (content=%d blocks, elapsed=%v, in=%d, out=%d)",
					len(partial.Content), time.Since(streamStart).Round(time.Millisecond), inputTokens, outputTokens)
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
			}
		}

		// Stream ended without message_stop.
		totalElapsed := time.Since(streamStart)
		if err := stream.Err(); err != nil {
			log.Printf("[anthropic-stream] stream error after %v (events=%d): %v",
				totalElapsed.Round(time.Millisecond), eventCount, err)
			errAMsg := &types.AssistantMessage{
				Role:         "assistant",
				Provider:     "anthropic",
				Model:        req.Model.ID,
				StopReason:   types.StopReasonError,
				ErrorMessage: err.Error(),
				Timestamp:    time.Now().UnixMilli(),
			}
			emit(types.AssistantMessageEvent{
				Type:  types.StreamEventError,
				Error: errAMsg,
			})
			return
		}

		log.Printf("[anthropic-stream] WARNING: SSE channel closed without message_stop (events=%d, content=%d blocks, stopReason=%q, elapsed=%v, in=%d, out=%d)",
			eventCount, len(partial.Content), partial.StopReason, totalElapsed.Round(time.Millisecond), inputTokens, outputTokens)
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
	}()
	return ch, nil
}

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

// buildTools converts ToolSchema slice to Anthropic SDK ToolUnionParam slice.
func buildTools(tools []provider.ToolSchema) []anthropicsdk.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropicsdk.ToolUnionParam, len(tools))
	for i, t := range tools {
		tp := anthropicsdk.ToolParam{
			Name:        t.Name,
			Description: anthropicsdk.String(t.Description),
			InputSchema: buildInputSchema(t.Parameters),
		}
		if i == len(tools)-1 {
			tp.CacheControl = anthropicsdk.NewCacheControlEphemeralParam()
		}
		out[i] = anthropicsdk.ToolUnionParam{OfTool: &tp}
	}
	return out
}

// buildInputSchema converts a raw JSON-schema map to ToolInputSchemaParam.
func buildInputSchema(schema map[string]any) anthropicsdk.ToolInputSchemaParam {
	p := anthropicsdk.ToolInputSchemaParam{}
	for k, v := range schema {
		switch k {
		case "type":
			// ToolInputSchemaParam already defaults to "object".
		case "properties":
			p.Properties = v
		case "required":
			if list, ok := v.([]any); ok {
				for _, r := range list {
					if s, ok := r.(string); ok {
						p.Required = append(p.Required, s)
					}
				}
			}
		default:
			if p.ExtraFields == nil {
				p.ExtraFields = make(map[string]any)
			}
			p.ExtraFields[k] = v
		}
	}
	return p
}

// convertMessages converts the internal message slice to Anthropic SDK MessageParam slice.
func convertMessages(msgs []types.Message) ([]anthropicsdk.MessageParam, error) {
	out := make([]anthropicsdk.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			blocks, err := convertBlocks(msg.Content)
			if err != nil {
				return nil, err
			}
			out = append(out, anthropicsdk.NewUserMessage(blocks...))
		case *types.AssistantMessage:
			blocks, err := convertBlocks(msg.Content)
			if err != nil {
				return nil, err
			}
			if len(blocks) == 0 {
				continue // Anthropic API rejects empty assistant messages.
			}
			out = append(out, anthropicsdk.NewAssistantMessage(blocks...))
		case *types.ToolResultMessage:
			content := ""
			if len(msg.Content) > 0 {
				if tc, ok := msg.Content[0].(*types.TextContent); ok {
					content = tc.Text
				}
			}
			out = append(out, anthropicsdk.NewUserMessage(
				anthropicsdk.NewToolResultBlock(msg.ToolCallID, content, msg.IsError),
			))
		}
	}
	return out, nil
}

// convertBlocks converts ContentBlock slice to ContentBlockParamUnion slice.
func convertBlocks(blocks []types.ContentBlock) ([]anthropicsdk.ContentBlockParamUnion, error) {
	out := make([]anthropicsdk.ContentBlockParamUnion, 0, len(blocks))
	for _, b := range blocks {
		switch block := b.(type) {
		case *types.TextContent:
			out = append(out, anthropicsdk.NewTextBlock(block.Text))
		case *types.ImageContent:
			if block.URL != "" {
				out = append(out, anthropicsdk.NewImageBlock(
					anthropicsdk.URLImageSourceParam{URL: block.URL},
				))
			} else {
				out = append(out, anthropicsdk.NewImageBlockBase64(block.MimeType, block.Data))
			}
		case *types.ToolCall:
			out = append(out, anthropicsdk.NewToolUseBlock(block.ID, block.Arguments, block.Name))
		case *types.ThinkingContent:
			if block.ThinkingSignature != "" {
				// Send back signature only; full thinking text is not needed for caching.
				out = append(out, anthropicsdk.NewThinkingBlock(block.ThinkingSignature, ""))
			}
		}
	}
	return out, nil
}

// injectCacheBreakpoints stamps cache_control on the last block of the second-to-last
// user message, so the entire conversation prefix is cached on subsequent turns.
func injectCacheBreakpoints(msgs []anthropicsdk.MessageParam) {
	var userIdx []int
	for i, m := range msgs {
		if m.Role == anthropicsdk.MessageParamRoleUser {
			userIdx = append(userIdx, i)
		}
	}
	if len(userIdx) < 2 {
		return
	}
	content := msgs[userIdx[len(userIdx)-2]].Content
	if len(content) == 0 {
		return
	}
	if last := content[len(content)-1]; last.OfText != nil {
		last.OfText.CacheControl = anthropicsdk.NewCacheControlEphemeralParam()
	}
}
