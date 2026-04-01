package provider

import (
	"strings"
)

// RetryDecision indicates how the agent loop should handle an error from prov.Stream().
type RetryDecision int

const (
	// RetryDecisionRetry means the error is transient and the request should be retried.
	RetryDecisionRetry RetryDecision = iota
	// RetryDecisionNoRetry means the error is permanent and should not be retried.
	RetryDecisionNoRetry
	// RetryDecisionNeedCompact means the prompt is too long and needs compaction before retry.
	RetryDecisionNeedCompact
)

// ClassifyError inspects an error and returns a RetryDecision.
func ClassifyError(err error) RetryDecision {
	if err == nil {
		return RetryDecisionNoRetry
	}

	// Check structured APIError first.
	if apiErr, ok := IsAPIError(err); ok {
		return classifyStatusCode(apiErr.StatusCode, apiErr.Body)
	}

	// Fall back to string matching for wrapped / network errors.
	msg := err.Error()
	if containsAny(msg, "connection reset", "connection refused", "eof", "broken pipe",
		"timeout", "tls handshake", "no such host") {
		return RetryDecisionRetry
	}

	return RetryDecisionNoRetry
}

func classifyStatusCode(code int, body string) RetryDecision {
	switch {
	// Rate limit or overloaded — always retry.
	case code == 429, code == 529, code == 502, code == 503:
		return RetryDecisionRetry

	// Bad request — check if it's a prompt-too-long error.
	case code == 400:
		lower := strings.ToLower(body)
		if containsAny(lower,
			"prompt is too long",
			"context_length_exceeded",
			"maximum context length",
			"too many tokens",
			"max_tokens",
			"content is too large",
		) {
			return RetryDecisionNeedCompact
		}
		return RetryDecisionNoRetry

	// Server error — retry.
	case code >= 500:
		return RetryDecisionRetry

	// Everything else (401, 403, 404, 413, etc.) — permanent.
	default:
		return RetryDecisionNoRetry
	}
}

func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}
