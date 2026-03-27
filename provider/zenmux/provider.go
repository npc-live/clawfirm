// Package zenmux implements an LLMProvider for ZenMux (https://zenmux.ai),
// an OpenAI-compatible large model aggregation platform.
//
// ZenMux aggregates models from OpenAI, Anthropic, Google, and others under
// a single API key. Model slugs use the format "provider/model-id", e.g.
// "openai/gpt-5" or "anthropic/claude-sonnet-4.5".
//
// Auth: ZENMUX_API_KEY environment variable or explicit key via New().
// Docs: https://zenmux.ai/docs/guide/quickstart.html
package zenmux

import (
	"context"
	"strings"

	"github.com/ai-gateway/pi-go/provider"
	"github.com/ai-gateway/pi-go/provider/openai"
	"github.com/ai-gateway/pi-go/types"
)

const (
	// DefaultBaseURL is the ZenMux OpenAI-compatible endpoint.
	DefaultBaseURL = "https://zenmux.ai/api/v1"

	// EnvKey is the environment variable name for the ZenMux API key.
	EnvKey = "ZENMUX_API_KEY"
)

// Provider wraps the OpenAI-compatible provider with ZenMux defaults.
type Provider struct {
	inner *openai.Provider
}

// New creates a ZenMux Provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{inner: openai.NewWithBaseURL(apiKey, DefaultBaseURL)}
}

// NewWithBaseURL creates a ZenMux Provider with a custom base URL (for testing).
func NewWithBaseURL(apiKey, baseURL string) *Provider {
	return &Provider{inner: openai.NewWithBaseURL(apiKey, baseURL)}
}

// ID returns "zenmux".
func (p *Provider) ID() string { return "zenmux" }

// Models returns the built-in ZenMux model list.
func (p *Provider) Models() []types.Model { return BuiltinModels() }

// Stream delegates to the inner OpenAI-compatible provider, overriding the
// provider field in the resulting AssistantMessage to "zenmux".
func (p *Provider) Stream(ctx context.Context, req provider.LLMRequest) (<-chan types.AssistantMessageEvent, error) {
	// Ensure the model's BaseURL is set to ZenMux so the inner provider uses
	// the right endpoint even if the caller left it blank.
	if req.Model.BaseURL == "" {
		req.Model.BaseURL = DefaultBaseURL
	}
	ch, err := p.inner.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	// Rewrite the provider field in done/error events so callers see "zenmux".
	out := make(chan types.AssistantMessageEvent, 32)
	go func() {
		defer close(out)
		for ev := range ch {
			if ev.Message != nil {
				cp := *ev.Message
				cp.Provider = "zenmux"
				ev.Message = &cp
			}
			if ev.Error != nil {
				cp := *ev.Error
				cp.Provider = "zenmux"
				ev.Error = &cp
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// IsQuotaRefreshError returns true when a ZenMux 402 response indicates a
// temporary subscription-window exhaustion that should be retried, rather than
// a permanent billing failure that should stop retries.
//
// ZenMux rolling-window quota resets automatically; callers should classify
// this as rate_limit (back off and retry) rather than billing (give up).
//
// Reference: https://zenmux.ai/docs/guide/subscription.html
func IsQuotaRefreshError(status int, message string) bool {
	if status != 402 {
		return false
	}
	return strings.Contains(message, "subscription quota limit") ||
		strings.Contains(message, "automatic quota refresh")
}
