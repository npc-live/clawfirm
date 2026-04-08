package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// KVStore provides generic JSON key-value persistence.
type KVStore struct{ db *DB }

// KV returns a KVStore backed by db.
func (d *DB) KV() *KVStore { return &KVStore{db: d} }

// Set stores value under key (upsert). value must be JSON-encodable.
func (s *KVStore) Set(key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("store: kv marshal: %w", err)
	}
	_, err = s.db.sql.Exec(
		`INSERT INTO kv(key, value, updated_at) VALUES(?,?,unixepoch())
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, string(b),
	)
	return err
}

// Get reads the JSON value for key into dst. Returns ErrNotFound if missing.
func (s *KVStore) Get(key string, dst any) error {
	var raw string
	err := s.db.sql.QueryRow(`SELECT value FROM kv WHERE key=?`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dst)
}

// Delete removes the key. No-op if not found.
func (s *KVStore) Delete(key string) error {
	_, err := s.db.sql.Exec(`DELETE FROM kv WHERE key=?`, key)
	return err
}

// ScanPrefix returns all key-value pairs where key starts with prefix.
// Values are raw JSON strings — callers should unmarshal as needed.
func (s *KVStore) ScanPrefix(prefix string) (map[string]string, error) {
	rows, err := s.db.sql.Query(`SELECT key, value FROM kv WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// ErrNotFound is returned by Get when the key does not exist.
var ErrNotFound = errors.New("store: key not found")
