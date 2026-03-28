package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-gateway/pi-go/message"
	"github.com/ai-gateway/pi-go/types"
)

// MessageRecord is a single raw message row from the database.
type MessageRecord struct {
	ID        int64
	ChannelID string
	UserID    string
	Role      string    // "user" | "assistant" | "tool"
	Content   string    // full message JSON (can be decoded with message.UnmarshalMessages)
	CreatedAt time.Time
}

// MessageStore handles conversation message persistence.
type MessageStore struct{ db *DB }

// Messages returns a MessageStore backed by db.
func (d *DB) Messages() *MessageStore { return &MessageStore{db: d} }

// SaveMessage persists a single types.Message (user / assistant / tool).
// The full message is JSON-serialised so it can be restored without data loss.
func (s *MessageStore) SaveMessage(channelID, userID string, msg types.Message) error {
	// MarshalMessages wraps in an array; we extract the single element.
	b, err := message.MarshalMessages([]types.Message{msg})
	if err != nil {
		return fmt.Errorf("store: marshal message: %w", err)
	}
	// Unwrap the outer array — store each message as its own JSON object.
	var arr []json.RawMessage
	if err := json.Unmarshal(b, &arr); err != nil || len(arr) == 0 {
		return fmt.Errorf("store: unwrap message JSON: %w", err)
	}
	role := roleOf(msg)
	_, err = s.db.sql.Exec(
		`INSERT INTO messages(channel_id, user_id, role, content) VALUES(?,?,?,?)`,
		channelID, userID, role, string(arr[0]),
	)
	return err
}

// ListMessages returns decoded types.Message values for a channel+user, ordered oldest-first.
// Default limit is 100 if Limit == 0.
func (s *MessageStore) ListMessages(p QueryParams) ([]types.Message, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.sql.Query(
		`SELECT content FROM messages
		 WHERE channel_id=? AND user_id=?
		 ORDER BY created_at ASC, id ASC
		 LIMIT ? OFFSET ?`,
		p.ChannelID, p.UserID, limit, p.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Message
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		// Wrap back into array for UnmarshalMessages
		wrapped := "[" + raw + "]"
		msgs, err := message.UnmarshalMessages([]byte(wrapped))
		if err != nil {
			return nil, fmt.Errorf("store: unmarshal message: %w", err)
		}
		out = append(out, msgs...)
	}
	return out, rows.Err()
}

// ListRecords returns raw MessageRecords (for inspection / debugging).
func (s *MessageStore) ListRecords(p QueryParams) ([]MessageRecord, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.sql.Query(
		`SELECT id, channel_id, user_id, role, content, created_at
		 FROM messages
		 WHERE channel_id=? AND user_id=?
		 ORDER BY created_at ASC, id ASC
		 LIMIT ? OFFSET ?`,
		p.ChannelID, p.UserID, limit, p.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MessageRecord
	for rows.Next() {
		var r MessageRecord
		var ts int64
		if err := rows.Scan(&r.ID, &r.ChannelID, &r.UserID, &r.Role, &r.Content, &ts); err != nil {
			return nil, err
		}
		r.CreatedAt = time.Unix(ts, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListUserIDs returns distinct user_id values for a channel, ordered by most recent activity.
func (s *MessageStore) ListUserIDs(channelID string) ([]string, error) {
	rows, err := s.db.sql.Query(
		`SELECT user_id FROM messages
		 WHERE channel_id=?
		 GROUP BY user_id
		 ORDER BY MAX(created_at) DESC, MAX(id) DESC`,
		channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CountByChannel returns the total number of messages for a channel (all users).
func (s *MessageStore) CountByChannel(channelID string) (int64, error) {
	var n int64
	err := s.db.sql.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE channel_id=?`, channelID,
	).Scan(&n)
	return n, err
}

// QueryParams controls pagination.
type QueryParams struct {
	ChannelID string
	UserID    string
	Limit     int
	Offset    int
}

func roleOf(m types.Message) string {
	switch m.(type) {
	case *types.AssistantMessage:
		return "assistant"
	case *types.ToolResultMessage:
		return "tool"
	default:
		return "user"
	}
}
