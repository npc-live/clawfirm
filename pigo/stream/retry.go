package stream

import (
	"context"
	"time"
)

// RetryConfig controls exponential backoff for stream reconnection.
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitialWait: 500 * time.Millisecond,
		MaxWait:     10 * time.Second,
		Multiplier:  2.0,
	}
}

// WithRetry calls fn up to cfg.MaxAttempts times, backing off between failures.
// If fn returns nil error, retry stops immediately.
func WithRetry(ctx context.Context, cfg RetryConfig, fn func(attempt int) error) error {
	wait := cfg.InitialWait
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn(attempt)
		if lastErr == nil {
			return nil
		}
		if attempt+1 >= cfg.MaxAttempts {
			break
		}
		// back off
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		wait = time.Duration(float64(wait) * cfg.Multiplier)
		if wait > cfg.MaxWait {
			wait = cfg.MaxWait
		}
	}
	return lastErr
}
