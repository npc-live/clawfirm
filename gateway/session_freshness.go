package gateway

import (
	"time"

	"github.com/ai-gateway/clawfirm/store"
)

// FreshnessConfig controls when a session is considered stale and should be reset.
type FreshnessConfig struct {
	Mode        store.ResetMode
	AtHour      int
	IdleMinutes int
}

// IsFresh reports whether a session is still fresh.
//
//   - never  → always true
//   - daily  → stale if lastUsed is before today's reset boundary AND the session
//     has not been reset since that boundary.
//   - idle   → stale if now − lastUsed > idleMinutes.
func IsFresh(entry *store.SessionEntry, cfg FreshnessConfig, now, lastUsed time.Time) bool {
	switch cfg.Mode {
	case store.ResetModeDaily:
		boundary := dailyResetBoundary(cfg.AtHour, now)
		if lastUsed.Before(boundary) {
			// Also check whether we already reset since the boundary.
			if entry.LastResetAt == nil || entry.LastResetAt.Before(boundary) {
				return false
			}
		}
		return true
	case store.ResetModeIdle:
		if cfg.IdleMinutes <= 0 {
			return true
		}
		return now.Sub(lastUsed) <= time.Duration(cfg.IdleMinutes)*time.Minute
	default: // ResetModeNever
		return true
	}
}

// dailyResetBoundary returns the most recent occurrence of h:00 UTC relative to now.
// If now is before today's h:00, it returns yesterday's h:00.
func dailyResetBoundary(h int, now time.Time) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, time.UTC)
	if now.UTC().Before(today) {
		return today.AddDate(0, 0, -1)
	}
	return today
}

// FreshnessConfigFromEntry builds a FreshnessConfig from a stored session entry.
func FreshnessConfigFromEntry(e *store.SessionEntry) FreshnessConfig {
	return FreshnessConfig{
		Mode:        e.ResetMode,
		AtHour:      e.ResetAtHour,
		IdleMinutes: e.IdleMinutes,
	}
}
