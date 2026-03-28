package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ScheduleData is the JSON payload stored in cron_jobs.schedule_data.
type ScheduleData struct {
	At       string `json:"at,omitempty"`
	EveryMs  int64  `json:"everyMs,omitempty"`
	AnchorMs int64  `json:"anchorMs,omitempty"`
	Expr     string `json:"expr,omitempty"`
	Tz       string `json:"tz,omitempty"`
}

// CronJob is a scheduled job persisted in SQLite.
type CronJob struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ScheduleKind string       `json:"scheduleKind"`
	Schedule     ScheduleData `json:"schedule"`
	AgentName    string       `json:"agentName"`
	Prompt       string       `json:"prompt"`
	Enabled      bool         `json:"enabled"`
	CreatedAt    *time.Time   `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time   `json:"updatedAt,omitempty"`
}

// CronJobHistory records a single execution of a cron job.
type CronJobHistory struct {
	ID         int64     `json:"id"`
	JobID      string    `json:"jobId"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	Status     string    `json:"status"` // "running", "success", "error"
	ResultText string    `json:"resultText"`
	ErrorText  string    `json:"errorText"`
}

// CronJobStore handles cron job persistence.
type CronJobStore struct{ db *DB }

// CronJobs returns a CronJobStore backed by db.
func (d *DB) CronJobs() *CronJobStore { return &CronJobStore{db: d} }

// Create inserts a new cron job. ID must be set by the caller.
func (s *CronJobStore) Create(job *CronJob) error {
	data, err := json.Marshal(job.Schedule)
	if err != nil {
		return fmt.Errorf("store: marshal schedule: %w", err)
	}
	enabled := 0
	if job.Enabled {
		enabled = 1
	}
	_, err = s.db.sql.Exec(
		`INSERT INTO cron_jobs(id, name, schedule_kind, schedule_data, agent_name, prompt, enabled)
		 VALUES(?,?,?,?,?,?,?)`,
		job.ID, job.Name, job.ScheduleKind, string(data), job.AgentName, job.Prompt, enabled,
	)
	return err
}

// Update modifies an existing cron job.
func (s *CronJobStore) Update(job *CronJob) error {
	data, err := json.Marshal(job.Schedule)
	if err != nil {
		return fmt.Errorf("store: marshal schedule: %w", err)
	}
	enabled := 0
	if job.Enabled {
		enabled = 1
	}
	_, err = s.db.sql.Exec(
		`UPDATE cron_jobs SET name=?, schedule_kind=?, schedule_data=?, agent_name=?, prompt=?, enabled=?, updated_at=unixepoch()
		 WHERE id=?`,
		job.Name, job.ScheduleKind, string(data), job.AgentName, job.Prompt, enabled, job.ID,
	)
	return err
}

// Delete removes a cron job by ID.
func (s *CronJobStore) Delete(id string) error {
	_, err := s.db.sql.Exec(`DELETE FROM cron_jobs WHERE id=?`, id)
	return err
}

// Get returns a single cron job by ID.
func (s *CronJobStore) Get(id string) (*CronJob, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, name, schedule_kind, schedule_data, agent_name, prompt, enabled, created_at, updated_at
		 FROM cron_jobs WHERE id=?`, id,
	)
	return scanCronJob(row)
}

// List returns all cron jobs.
func (s *CronJobStore) List() ([]CronJob, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, name, schedule_kind, schedule_data, agent_name, prompt, enabled, created_at, updated_at
		 FROM cron_jobs ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCronJobs(rows)
}

// ListEnabled returns only enabled cron jobs.
func (s *CronJobStore) ListEnabled() ([]CronJob, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, name, schedule_kind, schedule_data, agent_name, prompt, enabled, created_at, updated_at
		 FROM cron_jobs WHERE enabled=1 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCronJobs(rows)
}

// FindByName returns a cron job by name. Returns (nil, false, nil) if not found.
func (s *CronJobStore) FindByName(name string) (*CronJob, bool, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, name, schedule_kind, schedule_data, agent_name, prompt, enabled, created_at, updated_at
		 FROM cron_jobs WHERE name=?`, name,
	)
	job, err := scanCronJob(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return job, true, nil
}

// SetEnabled updates the enabled flag of a cron job.
func (s *CronJobStore) SetEnabled(id string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.sql.Exec(
		`UPDATE cron_jobs SET enabled=?, updated_at=unixepoch() WHERE id=?`, v, id,
	)
	return err
}

// InsertHistory creates a new history row with status "running" and returns its ID.
func (s *CronJobStore) InsertHistory(jobID string) (int64, error) {
	res, err := s.db.sql.Exec(
		`INSERT INTO cron_job_history(job_id, started_at, status) VALUES(?,unixepoch(),'running')`,
		jobID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CompleteHistory finalises a history row.
func (s *CronJobStore) CompleteHistory(historyID int64, status, resultText, errorText string) error {
	_, err := s.db.sql.Exec(
		`UPDATE cron_job_history SET finished_at=unixepoch(), status=?, result_text=?, error_text=? WHERE id=?`,
		status, resultText, errorText, historyID,
	)
	return err
}

// ListHistory returns recent history rows for a specific job.
func (s *CronJobStore) ListHistory(jobID string, limit int) ([]CronJobHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.sql.Query(
		`SELECT id, job_id, started_at, COALESCE(finished_at,0), status, result_text, error_text
		 FROM cron_job_history WHERE job_id=? ORDER BY started_at DESC LIMIT ?`,
		jobID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistoryRows(rows)
}

// ListHistoryAll returns recent history rows across all jobs.
func (s *CronJobStore) ListHistoryAll(limit int) ([]CronJobHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.sql.Query(
		`SELECT id, job_id, started_at, COALESCE(finished_at,0), status, result_text, error_text
		 FROM cron_job_history ORDER BY started_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistoryRows(rows)
}

// --- internal scanners ---

type scannable interface {
	Scan(dest ...any) error
}

func scanCronJob(row scannable) (*CronJob, error) {
	var j CronJob
	var data string
	var enabled int
	var createdAt, updatedAt int64
	err := row.Scan(&j.ID, &j.Name, &j.ScheduleKind, &data, &j.AgentName, &j.Prompt, &enabled, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	j.Enabled = enabled != 0
	ca := time.Unix(createdAt, 0)
	ua := time.Unix(updatedAt, 0)
	j.CreatedAt = &ca
	j.UpdatedAt = &ua
	if err := json.Unmarshal([]byte(data), &j.Schedule); err != nil {
		return nil, fmt.Errorf("store: unmarshal schedule: %w", err)
	}
	return &j, nil
}

func scanCronJobs(rows *sql.Rows) ([]CronJob, error) {
	var out []CronJob
	for rows.Next() {
		j, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func scanHistoryRows(rows *sql.Rows) ([]CronJobHistory, error) {
	var out []CronJobHistory
	for rows.Next() {
		var h CronJobHistory
		var startedAt, finishedAt int64
		if err := rows.Scan(&h.ID, &h.JobID, &startedAt, &finishedAt, &h.Status, &h.ResultText, &h.ErrorText); err != nil {
			return nil, err
		}
		h.StartedAt = time.Unix(startedAt, 0)
		if finishedAt > 0 {
			h.FinishedAt = time.Unix(finishedAt, 0)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
