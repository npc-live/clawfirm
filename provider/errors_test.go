package provider

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "with body",
			err:  &APIError{StatusCode: 429, Body: "rate limited", Provider: "anthropic"},
			want: "anthropic: HTTP 429: rate limited",
		},
		{
			name: "without body",
			err:  &APIError{StatusCode: 500, Provider: "openai"},
			want: "openai: HTTP 500",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsAPIError(t *testing.T) {
	apiErr := &APIError{StatusCode: 429, Provider: "test"}
	wrapped := fmt.Errorf("wrapped: %w", apiErr)

	got, ok := IsAPIError(wrapped)
	if !ok {
		t.Fatal("expected to find APIError in wrapped error")
	}
	if got.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", got.StatusCode)
	}

	_, ok = IsAPIError(fmt.Errorf("plain error"))
	if ok {
		t.Fatal("expected no APIError from plain error")
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Run("integer seconds", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "30")
		d := ParseRetryAfter(resp)
		if d != 30*time.Second {
			t.Errorf("got %v, want 30s", d)
		}
	})

	t.Run("absent header", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		d := ParseRetryAfter(resp)
		if d != 0 {
			t.Errorf("got %v, want 0", d)
		}
	})
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want RetryDecision
	}{
		{
			name: "nil",
			err:  nil,
			want: RetryDecisionNoRetry,
		},
		{
			name: "429 rate limit",
			err:  &APIError{StatusCode: 429, Provider: "anthropic"},
			want: RetryDecisionRetry,
		},
		{
			name: "529 overloaded",
			err:  &APIError{StatusCode: 529, Provider: "anthropic"},
			want: RetryDecisionRetry,
		},
		{
			name: "502 bad gateway",
			err:  &APIError{StatusCode: 502, Provider: "openai"},
			want: RetryDecisionRetry,
		},
		{
			name: "503 unavailable",
			err:  &APIError{StatusCode: 503, Provider: "openai"},
			want: RetryDecisionRetry,
		},
		{
			name: "500 server error",
			err:  &APIError{StatusCode: 500, Provider: "gemini"},
			want: RetryDecisionRetry,
		},
		{
			name: "401 unauthorized",
			err:  &APIError{StatusCode: 401, Provider: "anthropic"},
			want: RetryDecisionNoRetry,
		},
		{
			name: "403 forbidden",
			err:  &APIError{StatusCode: 403, Provider: "openai"},
			want: RetryDecisionNoRetry,
		},
		{
			name: "400 prompt too long",
			err:  &APIError{StatusCode: 400, Body: `{"error":{"message":"prompt is too long"}}`, Provider: "anthropic"},
			want: RetryDecisionNeedCompact,
		},
		{
			name: "400 context length exceeded",
			err:  &APIError{StatusCode: 400, Body: `{"error":{"message":"This model's maximum context length is exceeded"}}`, Provider: "openai"},
			want: RetryDecisionNeedCompact,
		},
		{
			name: "400 other",
			err:  &APIError{StatusCode: 400, Body: `{"error":{"message":"invalid request"}}`, Provider: "anthropic"},
			want: RetryDecisionNoRetry,
		},
		{
			name: "connection reset (string match)",
			err:  fmt.Errorf("read tcp: connection reset by peer"),
			want: RetryDecisionRetry,
		},
		{
			name: "EOF (string match)",
			err:  fmt.Errorf("unexpected EOF"),
			want: RetryDecisionRetry,
		},
		{
			name: "timeout (string match)",
			err:  fmt.Errorf("context deadline exceeded (Client.Timeout)"),
			want: RetryDecisionRetry,
		},
		{
			name: "wrapped APIError 429",
			err:  fmt.Errorf("stream failed: %w", &APIError{StatusCode: 429, Provider: "test"}),
			want: RetryDecisionRetry,
		},
		{
			name: "plain error",
			err:  fmt.Errorf("something went wrong"),
			want: RetryDecisionNoRetry,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if got != tt.want {
				t.Errorf("ClassifyError() = %d, want %d", got, tt.want)
			}
		})
	}
}
