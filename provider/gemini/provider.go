package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/stream"
	"github.com/ai-gateway/pi-go/types"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Provider implements provider.LLMProvider for Google's Gemini API.
type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// New creates a Gemini Provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// NewWithBaseURL creates a Gemini Provider with a custom base URL (for testing).
func NewWithBaseURL(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// ID returns "gemini".
func (p *Provider) ID() string { return "gemini" }

// Models returns the built-in Gemini model list.
func (p *Provider) Models() []types.Model { return BuiltinModels() }

// --- Gemini API types ---

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
	InlineData   *geminiInlineData   `json:"inlineData,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type geminiRequest struct {
	Contents         []geminiContent    `json:"contents"`
	SystemInstruction *geminiContent    `json:"systemInstruction,omitempty"`
	Tools            []geminiTool       `json:"tools,omitempty"`
	GenerationConfig *geminiGenConfig   `json:"generationConfig,omitempty"`
}

type geminiGenConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

// Stream sends a streaming request to Gemini and returns an event channel.
func (p *Provider) Stream(ctx context.Context, req provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	apiKey := req.Options.APIKey
	if apiKey == "" {
		apiKey = p.apiKey
	}

	contents := convertMessages(req.Messages)

	var sysInstruction *geminiContent
	if req.SystemPrompt != "" {
		sysInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	var tools []geminiTool
	if len(req.Tools) > 0 {
		var decls []geminiFunctionDecl
		for _, t := range req.Tools {
			decls = append(decls, geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		tools = []geminiTool{{FunctionDeclarations: decls}}
	}

	genConfig := &geminiGenConfig{}
	if req.Model.MaxTokens > 0 {
		genConfig.MaxOutputTokens = req.Model.MaxTokens
	}
	if req.Options.MaxTokens != nil {
		genConfig.MaxOutputTokens = *req.Options.MaxTokens
	}
	if req.Options.Temperature != nil {
		genConfig.Temperature = req.Options.Temperature
	}

	body := geminiRequest{
		Contents:         contents,
		SystemInstruction: sysInstruction,
		Tools:            tools,
		GenerationConfig: genConfig,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	baseURL := p.baseURL
	if req.Model.BaseURL != "" {
		baseURL = req.Model.BaseURL
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
		baseURL, req.Model.ID, apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Options.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: http request: %w", err)
	}

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(b))
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
		Provider: "gemini",
		Model:    modelID,
	}

	var inputTokens, outputTokens int
	contentIndex := 0

	sseEvents := stream.ParseSSEStream(ctx, body)
	for sseEv := range sseEvents {
		if sseEv.Data == "" {
			continue
		}

		var gr geminiResponse
		if err := json.Unmarshal([]byte(sseEv.Data), &gr); err != nil {
			continue
		}

		if gr.UsageMetadata != nil {
			inputTokens = gr.UsageMetadata.PromptTokenCount
			outputTokens = gr.UsageMetadata.CandidatesTokenCount
		}

		for _, cand := range gr.Candidates {
			finishReason := cand.FinishReason

			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventTextStart,
						ContentIndex: contentIndex,
					})
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventTextDelta,
						ContentIndex: contentIndex,
						Delta:        part.Text,
					})
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventTextEnd,
						ContentIndex: contentIndex,
						Content:      part.Text,
					})
					partial.Content = append(partial.Content, &types.TextContent{
						Type: types.ContentTypeText,
						Text: part.Text,
					})
					contentIndex++
				}

				if part.FunctionCall != nil {
					tc := &types.ToolCall{
						Type:      types.ContentTypeToolCall,
						ID:        "call_" + part.FunctionCall.Name,
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					}
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventToolCallStart,
						ContentIndex: contentIndex,
						ToolCall:     tc,
					})
					emit(types.AssistantMessageEvent{
						Type:         types.StreamEventToolCallEnd,
						ContentIndex: contentIndex,
						ToolCall:     tc,
					})
					partial.Content = append(partial.Content, tc)
					contentIndex++
				}
			}

			if finishReason != "" {
				partial.StopReason = mapStopReason(finishReason, partial.Content)
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

// mapStopReason maps Gemini finish reasons to canonical StopReason.
func mapStopReason(reason string, content []types.ContentBlock) types.StopReason {
	switch reason {
	case "STOP":
		// Check if we have any tool calls
		for _, c := range content {
			if c.ContentType() == types.ContentTypeToolCall {
				return types.StopReasonToolUse
			}
		}
		return types.StopReasonStop
	case "MAX_TOKENS":
		return types.StopReasonLength
	case "FUNCTION_CALL":
		return types.StopReasonToolUse
	default:
		return types.StopReasonStop
	}
}

// convertMessages converts types.Message slice to Gemini content format.
func convertMessages(msgs []types.Message) []geminiContent {
	var out []geminiContent
	for _, m := range msgs {
		switch msg := m.(type) {
		case *types.UserMessage:
			var parts []geminiPart
			for _, c := range msg.Content {
				switch block := c.(type) {
				case *types.TextContent:
					parts = append(parts, geminiPart{Text: block.Text})
				case *types.ImageContent:
					parts = append(parts, geminiPart{
						InlineData: &geminiInlineData{
							MimeType: block.MimeType,
							Data:     block.Data,
						},
					})
				}
			}
			out = append(out, geminiContent{Role: "user", Parts: parts})
		case *types.AssistantMessage:
			var parts []geminiPart
			for _, c := range msg.Content {
				switch block := c.(type) {
				case *types.TextContent:
					parts = append(parts, geminiPart{Text: block.Text})
				case *types.ToolCall:
					parts = append(parts, geminiPart{
						FunctionCall: &geminiFunctionCall{
							Name: block.Name,
							Args: block.Arguments,
						},
					})
				}
			}
			out = append(out, geminiContent{Role: "model", Parts: parts})
		case *types.ToolResultMessage:
			var text string
			if len(msg.Content) > 0 {
				if tc, ok := msg.Content[0].(*types.TextContent); ok {
					text = tc.Text
				}
			}
			parts := []geminiPart{{
				FunctionResponse: &geminiFunctionResponse{
					Name:     msg.ToolName,
					Response: map[string]any{"result": text},
				},
			}}
			out = append(out, geminiContent{Role: "user", Parts: parts})
		}
	}
	return out
}
