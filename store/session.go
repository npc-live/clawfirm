package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ResetMode controls when a session's message history is cleared.
type ResetMode string

const (
	ResetModeNever ResetMode = "never"
	ResetModeDaily ResetMode = "daily"
	ResetModeIdle  ResetMode = "idle"
)

// SessionOrigin holds routing metadata about where a session originated.
type SessionOrigin struct {
	Label     string `json:"label,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Surface   string `json:"surface,omitempty"`
	ChatType  string `json:"chatType,omitempty"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	AccountID string `json:"accountId,omitempty"`
	ThreadID  string `json:"threadId,omitempty"`
}

// SessionEntry is a persisted session record.
type SessionEntry struct {
	SessionKey  string
	SessionID   string
	ChannelID   string
	UserID      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Label       string
	DisplayName string
	Subject     string
	ChatType    string
	Origin      *SessionOrigin

	LastChannel   string
	LastTo        string
	LastAccountID string
	LastThreadID  string

	InputTokens      int
	OutputTokens     int
	TotalTokens      int
	CacheRead        int
	CacheWrite       int
	EstimatedCostUSD float64
	Model            string
	ModelProvider    string
	CompactionCount  int

	EndedAt     *time.Time
	SystemSent  bool
	LastResetAt *time.Time
	ResetMode   ResetMode
	ResetAtHour int
	IdleMinutes int
}

// UsageDelta is applied atomically to a session's token counters.
type UsageDelta struct {
	InputTokens      int
	OutputTokens     int
	CacheRead        int
	CacheWrite       int
	EstimatedCostUSD float64
	Model            string
	ModelProvider    string
}

// SessionStore handles session persistence.
type SessionStore struct{ db *DB }

// Sessions returns a SessionStore backed by db.
func (d *DB) Sessions() *SessionStore { return &SessionStore{db: d} }

// Upsert inserts or replaces a session record.
func (s *SessionStore) Upsert(e *SessionEntry) error {
	if e.SessionID == "" {
		e.SessionID = uuid.New().String()
	}
	originJSON, err := marshalOrigin(e.Origin)
	if err != nil {
		return fmt.Errorf("store/session: marshal origin: %w", err)
	}
	var endedAt, lastResetAt *int64
	if e.EndedAt != nil {
		ms := e.EndedAt.UnixMilli()
		endedAt = &ms
	}
	if e.LastResetAt != nil {
		ms := e.LastResetAt.UnixMilli()
		lastResetAt = &ms
	}
	systemSent := 0
	if e.SystemSent {
		systemSent = 1
	}
	_, err = s.db.sql.Exec(`
		INSERT OR REPLACE INTO sessions (
			session_key, session_id, channel_id, user_id,
			created_at, updated_at,
			label, display_name, subject, chat_type, origin_json,
			last_channel, last_to, last_account_id, last_thread_id,
			input_tokens, output_tokens, total_tokens,
			cache_read, cache_write, estimated_cost_usd,
			model, model_provider, compaction_count,
			ended_at, system_sent, last_reset_at,
			reset_mode, reset_at_hour, idle_minutes
		) VALUES (
			?,?,?,?,
			?,?,
			?,?,?,?,?,
			?,?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,?
		)`,
		e.SessionKey, e.SessionID, e.ChannelID, e.UserID,
		e.CreatedAt.UnixMilli(), e.UpdatedAt.UnixMilli(),
		e.Label, e.DisplayName, e.Subject, e.ChatType, originJSON,
		e.LastChannel, e.LastTo, e.LastAccountID, e.LastThreadID,
		e.InputTokens, e.OutputTokens, e.TotalTokens,
		e.CacheRead, e.CacheWrite, e.EstimatedCostUSD,
		e.Model, e.ModelProvider, e.CompactionCount,
		endedAt, systemSent, lastResetAt,
		string(e.ResetMode), e.ResetAtHour, e.IdleMinutes,
	)
	return err
}

// Get returns the session for a given session_key, or nil if not found.
func (s *SessionStore) Get(sessionKey string) (*SessionEntry, error) {
	row := s.db.sql.QueryRow(`SELECT `+sessionCols+` FROM sessions WHERE session_key=?`, sessionKey)
	return scanSession(row)
}

// GetBySessionID returns the session for a given session_id, or nil if not found.
func (s *SessionStore) GetBySessionID(sessionID string) (*SessionEntry, error) {
	row := s.db.sql.QueryRow(`SELECT `+sessionCols+` FROM sessions WHERE session_id=?`, sessionID)
	return scanSession(row)
}

// List returns all sessions for a channel+user ordered by updated_at DESC.
func (s *SessionStore) List(channelID, userID string) ([]SessionEntry, error) {
	rows, err := s.db.sql.Query(
		`SELECT `+sessionCols+` FROM sessions WHERE channel_id=? AND user_id=? ORDER BY updated_at DESC`,
		channelID, userID,
	)
	if err != nil {
		return nil, err
	}
	return scanSessions(rows)
}

// ListByAgent returns all sessions whose session_key starts with "agent:{name}:".
func (s *SessionStore) ListByAgent(agentName string) ([]SessionEntry, error) {
	prefix := "agent:" + agentName + ":%"
	rows, err := s.db.sql.Query(
		`SELECT `+sessionCols+` FROM sessions WHERE session_key LIKE ? ORDER BY updated_at DESC`,
		prefix,
	)
	if err != nil {
		return nil, err
	}
	return scanSessions(rows)
}

// UpdateUsage atomically adds token/cost deltas to a session.
func (s *SessionStore) UpdateUsage(sessionKey string, u UsageDelta) error {
	_, err := s.db.sql.Exec(`
		UPDATE sessions SET
			input_tokens       = input_tokens + ?,
			output_tokens      = output_tokens + ?,
			total_tokens       = total_tokens + ? + ?,
			cache_read         = cache_read + ?,
			cache_write        = cache_write + ?,
			estimated_cost_usd = estimated_cost_usd + ?,
			model              = CASE WHEN ? != '' THEN ? ELSE model END,
			model_provider     = CASE WHEN ? != '' THEN ? ELSE model_provider END,
			updated_at         = (unixepoch()*1000)
		WHERE session_key = ?`,
		u.InputTokens,
		u.OutputTokens,
		u.InputTokens, u.OutputTokens,
		u.CacheRead,
		u.CacheWrite,
		u.EstimatedCostUSD,
		u.Model, u.Model,
		u.ModelProvider, u.ModelProvider,
		sessionKey,
	)
	return err
}

// MarkEnded sets ended_at to now for the given session.
func (s *SessionStore) MarkEnded(sessionKey string) error {
	_, err := s.db.sql.Exec(
		`UPDATE sessions SET ended_at=(unixepoch()*1000), updated_at=(unixepoch()*1000) WHERE session_key=?`,
		sessionKey,
	)
	return err
}

// MarkReset sets last_reset_at to now for the given session.
func (s *SessionStore) MarkReset(sessionKey string) error {
	_, err := s.db.sql.Exec(
		`UPDATE sessions SET last_reset_at=(unixepoch()*1000), updated_at=(unixepoch()*1000) WHERE session_key=?`,
		sessionKey,
	)
	return err
}

// ── internal helpers ─────────────────────────────────────────────────────────

const sessionCols = `session_key, session_id, channel_id, user_id,
	created_at, updated_at,
	label, display_name, subject, chat_type, origin_json,
	last_channel, last_to, last_account_id, last_thread_id,
	input_tokens, output_tokens, total_tokens,
	cache_read, cache_write, estimated_cost_usd,
	model, model_provider, compaction_count,
	ended_at, system_sent, last_reset_at,
	reset_mode, reset_at_hour, idle_minutes`

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanSession(row sessionScanner) (*SessionEntry, error) {
	var e SessionEntry
	var createdMS, updatedMS int64
	var originJSON string
	var endedMS, resetMS sql.NullInt64
	var systemSent int
	var resetMode string
	err := row.Scan(
		&e.SessionKey, &e.SessionID, &e.ChannelID, &e.UserID,
		&createdMS, &updatedMS,
		&e.Label, &e.DisplayName, &e.Subject, &e.ChatType, &originJSON,
		&e.LastChannel, &e.LastTo, &e.LastAccountID, &e.LastThreadID,
		&e.InputTokens, &e.OutputTokens, &e.TotalTokens,
		&e.CacheRead, &e.CacheWrite, &e.EstimatedCostUSD,
		&e.Model, &e.ModelProvider, &e.CompactionCount,
		&endedMS, &systemSent, &resetMS,
		&resetMode, &e.ResetAtHour, &e.IdleMinutes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.CreatedAt = time.UnixMilli(createdMS)
	e.UpdatedAt = time.UnixMilli(updatedMS)
	e.SystemSent = systemSent != 0
	e.ResetMode = ResetMode(resetMode)
	if endedMS.Valid {
		t := time.UnixMilli(endedMS.Int64)
		e.EndedAt = &t
	}
	if resetMS.Valid {
		t := time.UnixMilli(resetMS.Int64)
		e.LastResetAt = &t
	}
	if originJSON != "" && originJSON != "{}" {
		var origin SessionOrigin
		if err := json.Unmarshal([]byte(originJSON), &origin); err == nil {
			e.Origin = &origin
		}
	}
	return &e, nil
}

func scanSessions(rows *sql.Rows) ([]SessionEntry, error) {
	defer rows.Close()
	var out []SessionEntry
	for rows.Next() {
		e, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		if e != nil {
			out = append(out, *e)
		}
	}
	return out, rows.Err()
}

func marshalOrigin(o *SessionOrigin) (string, error) {
	if o == nil {
		return "{}", nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}
