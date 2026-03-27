package runtime

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// RunRecord represents a single execution run persisted in the state store.
type RunRecord struct {
	ID         int64
	FilePath   string
	Status     string // "running", "completed", "failed"
	StartedAt  int64  // Unix ms
	FinishedAt *int64 // nullable
	ErrorMsg   *string // nullable
}

// SessionRecord represents a single session result persisted in the state store.
type SessionRecord struct {
	ID            int64
	RunID         int64
	SessionIndex  int
	Prompt        string
	Output        string
	Model         string
	DurationMs    int64
	TokensUsed    *int    // nullable
	ToolCallsJSON *string // nullable
	VariablesJSON string
	CompletedAt   int64
}

// StateStore provides persistent storage for run and session state using SQLite.
type StateStore struct {
	db *sql.DB
}

// NewStateStore creates a new StateStore backed by a SQLite database at dbPath.
// It creates the parent directory if needed, opens the database, configures
// WAL journal mode and foreign keys, and initializes the schema.
func NewStateStore(dbPath string) (*StateStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("state_store: create directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("state_store: open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("state_store: set journal_mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("state_store: set foreign_keys: %w", err)
	}

	s := &StateStore{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("state_store: init schema: %w", err)
	}

	return s, nil
}

// initSchema creates the required tables if they do not already exist.
func (s *StateStore) initSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS runs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path   TEXT NOT NULL,
    status      TEXT NOT NULL,
    started_at  INTEGER NOT NULL,
    finished_at INTEGER,
    error_msg   TEXT
);
CREATE TABLE IF NOT EXISTS sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id          INTEGER NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    session_index   INTEGER NOT NULL,
    prompt          TEXT NOT NULL DEFAULT '',
    output          TEXT NOT NULL,
    model           TEXT NOT NULL,
    duration_ms     INTEGER NOT NULL,
    tokens_used     INTEGER,
    tool_calls_json TEXT,
    variables_json  TEXT NOT NULL,
    completed_at    INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS user_inputs (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id   INTEGER NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    var_name TEXT NOT NULL,
    answer   TEXT NOT NULL,
    UNIQUE(run_id, var_name)
);`
	_, err := s.db.Exec(schema)
	return err
}

// StartRun inserts a new run record with status "running" and returns the new run ID.
func (s *StateStore) StartRun(filePath string) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := s.db.Exec(
		"INSERT INTO runs (file_path, status, started_at) VALUES (?, ?, ?)",
		filePath, "running", now,
	)
	if err != nil {
		return 0, fmt.Errorf("state_store: start run: %w", err)
	}
	return res.LastInsertId()
}

// CompleteRun marks the given run as completed and sets its finished_at timestamp.
func (s *StateStore) CompleteRun(runID int64) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		"UPDATE runs SET status = ?, finished_at = ? WHERE id = ?",
		"completed", now, runID,
	)
	if err != nil {
		return fmt.Errorf("state_store: complete run: %w", err)
	}
	return nil
}

// FailRun marks the given run as failed with the provided error message and
// sets its finished_at timestamp.
func (s *StateStore) FailRun(runID int64, errorMsg string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		"UPDATE runs SET status = ?, finished_at = ?, error_msg = ? WHERE id = ?",
		"failed", now, errorMsg, runID,
	)
	if err != nil {
		return fmt.Errorf("state_store: fail run: %w", err)
	}
	return nil
}

// RecordSession inserts a session record for the given run. The variables map
// and tool calls slice are serialized to JSON.
func (s *StateStore) RecordSession(runID int64, sessionIndex int, prompt string, result *SessionResult, variables map[string]any) error {
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return fmt.Errorf("state_store: marshal variables: %w", err)
	}

	var toolCallsJSON *string
	if len(result.Metadata.ToolCalls) > 0 {
		b, err := json.Marshal(result.Metadata.ToolCalls)
		if err != nil {
			return fmt.Errorf("state_store: marshal tool_calls: %w", err)
		}
		str := string(b)
		toolCallsJSON = &str
	}

	var tokensUsed *int
	if result.Metadata.TokensUsed > 0 {
		v := result.Metadata.TokensUsed
		tokensUsed = &v
	}

	now := time.Now().UnixMilli()
	_, err = s.db.Exec(
		`INSERT INTO sessions
			(run_id, session_index, prompt, output, model, duration_ms, tokens_used, tool_calls_json, variables_json, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID,
		sessionIndex,
		prompt,
		result.Output,
		result.Metadata.Model,
		result.Metadata.Duration,
		tokensUsed,
		toolCallsJSON,
		string(variablesJSON),
		now,
	)
	if err != nil {
		return fmt.Errorf("state_store: record session: %w", err)
	}
	return nil
}

// GetCompletedSessions returns all session records for the given run, ordered
// by session_index ascending.
func (s *StateStore) GetCompletedSessions(runID int64) ([]SessionRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, session_index, prompt, output, model, duration_ms,
		        tokens_used, tool_calls_json, variables_json, completed_at
		 FROM sessions
		 WHERE run_id = ?
		 ORDER BY session_index ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("state_store: get completed sessions: %w", err)
	}
	defer rows.Close()

	var records []SessionRecord
	for rows.Next() {
		var r SessionRecord
		if err := rows.Scan(
			&r.ID,
			&r.RunID,
			&r.SessionIndex,
			&r.Prompt,
			&r.Output,
			&r.Model,
			&r.DurationMs,
			&r.TokensUsed,
			&r.ToolCallsJSON,
			&r.VariablesJSON,
			&r.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("state_store: scan session row: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state_store: iterate session rows: %w", err)
	}
	return records, nil
}

// FindIncompleteRun returns the most recent run for the given file path that
// has status "running" or "failed". Returns nil if no such run exists.
func (s *StateStore) FindIncompleteRun(filePath string) (*RunRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, file_path, status, started_at, finished_at, error_msg
		 FROM runs
		 WHERE file_path = ? AND status IN ('running', 'failed')
		 ORDER BY id DESC
		 LIMIT 1`,
		filePath,
	)

	var r RunRecord
	err := row.Scan(&r.ID, &r.FilePath, &r.Status, &r.StartedAt, &r.FinishedAt, &r.ErrorMsg)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state_store: find incomplete run: %w", err)
	}
	return &r, nil
}

// SaveUserInput persists a user-provided input variable for the given run.
// If the variable already exists for that run, its answer is updated.
func (s *StateStore) SaveUserInput(runID int64, varName, answer string) error {
	_, err := s.db.Exec(
		`INSERT INTO user_inputs (run_id, var_name, answer) VALUES (?, ?, ?)
		 ON CONFLICT(run_id, var_name) DO UPDATE SET answer = excluded.answer`,
		runID, varName, answer,
	)
	if err != nil {
		return fmt.Errorf("state_store: save user input: %w", err)
	}
	return nil
}

// GetUserInputs returns all saved user inputs for the given run as a map
// from variable name to answer.
func (s *StateStore) GetUserInputs(runID int64) (map[string]string, error) {
	rows, err := s.db.Query(
		"SELECT var_name, answer FROM user_inputs WHERE run_id = ?",
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("state_store: get user inputs: %w", err)
	}
	defer rows.Close()

	inputs := make(map[string]string)
	for rows.Next() {
		var name, answer string
		if err := rows.Scan(&name, &answer); err != nil {
			return nil, fmt.Errorf("state_store: scan user input row: %w", err)
		}
		inputs[name] = answer
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state_store: iterate user input rows: %w", err)
	}
	return inputs, nil
}

// Close closes the underlying database connection.
func (s *StateStore) Close() error {
	return s.db.Close()
}
