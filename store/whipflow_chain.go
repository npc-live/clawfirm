package store

import "time"

// WhipflowChainEntry is a single node in the whipflow execution chain.
type WhipflowChainEntry struct {
	ID         int64  `json:"id"`
	ChannelID  string `json:"channelId"`
	UserID     string `json:"userId"`
	CallID     string `json:"callId"`
	ParentID   string `json:"parentId,omitempty"`
	SessionIdx int    `json:"sessionIdx"` // -1 = preview
	Status     string `json:"status"`     // running | done | error
	CreatedAt  int64  `json:"createdAt"`
}

// WhipflowChainStore provides CRUD for the whipflow_chain table.
type WhipflowChainStore struct{ db *DB }

// WhipflowChain returns the whipflow chain store accessor.
func (d *DB) WhipflowChain() *WhipflowChainStore { return &WhipflowChainStore{db: d} }

// Insert adds a new chain entry.
func (s *WhipflowChainStore) Insert(e WhipflowChainEntry) error {
	if e.CreatedAt == 0 {
		e.CreatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.sql.Exec(
		`INSERT INTO whipflow_chain(channel_id, user_id, call_id, parent_id, session_idx, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ChannelID, e.UserID, e.CallID, e.ParentID, e.SessionIdx, e.Status, e.CreatedAt,
	)
	return err
}

// UpdateStatus updates the status of a chain entry by call_id.
func (s *WhipflowChainStore) UpdateStatus(callID, status string) error {
	_, err := s.db.sql.Exec(`UPDATE whipflow_chain SET status = ? WHERE call_id = ?`, status, callID)
	return err
}

// ListBySession returns all chain entries for a channel+user, ordered by creation time.
func (s *WhipflowChainStore) ListBySession(channelID, userID string) ([]WhipflowChainEntry, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, channel_id, user_id, call_id, COALESCE(parent_id,''), session_idx, status, created_at
		 FROM whipflow_chain
		 WHERE channel_id = ? AND user_id = ?
		 ORDER BY created_at ASC`,
		channelID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WhipflowChainEntry
	for rows.Next() {
		var e WhipflowChainEntry
		if err := rows.Scan(&e.ID, &e.ChannelID, &e.UserID, &e.CallID, &e.ParentID, &e.SessionIdx, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetByCallID returns a single chain entry by call_id.
func (s *WhipflowChainStore) GetByCallID(callID string) (*WhipflowChainEntry, error) {
	var e WhipflowChainEntry
	err := s.db.sql.QueryRow(
		`SELECT id, channel_id, user_id, call_id, COALESCE(parent_id,''), session_idx, status, created_at
		 FROM whipflow_chain WHERE call_id = ?`, callID,
	).Scan(&e.ID, &e.ChannelID, &e.UserID, &e.CallID, &e.ParentID, &e.SessionIdx, &e.Status, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// LatestCallID returns the most recent call_id for a channel+user, or "" if none.
func (s *WhipflowChainStore) LatestCallID(channelID, userID string) string {
	var callID string
	_ = s.db.sql.QueryRow(
		`SELECT call_id FROM whipflow_chain WHERE channel_id = ? AND user_id = ? ORDER BY created_at DESC LIMIT 1`,
		channelID, userID,
	).Scan(&callID)
	return callID
}
