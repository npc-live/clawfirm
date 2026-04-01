package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// APIError is a structured error returned by LLM providers on HTTP failures.
type APIError struct {
	StatusCode int
	Body       string
	Provider   string
	RetryAfter time.Duration // parsed from Retry-After header, 0 if absent
}

func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%s: HTTP %d: %s", e.Provider, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("%s: HTTP %d", e.Provider, e.StatusCode)
}

// IsAPIError extracts an *APIError from err using errors.As.
func IsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// ParseRetryAfter extracts the Retry-After header value as a Duration.
// Supports both integer seconds and HTTP-date formats.
// Returns 0 if the header is absent or unparseable.
func ParseRetryAfter(resp *http.Response) time.Duration {
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return 0
	}
	// Try integer seconds first.
	if secs, err := strconv.Atoi(val); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	// Try HTTP-date format.
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
