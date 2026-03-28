-- Cron job scheduler tables

CREATE TABLE IF NOT EXISTS cron_jobs (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    schedule_kind   TEXT NOT NULL,              -- 'at', 'every', 'cron'
    schedule_data   TEXT NOT NULL DEFAULT '{}', -- JSON: {at, everyMs, anchorMs, expr, tz}
    agent_name      TEXT NOT NULL,
    prompt          TEXT NOT NULL,
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS cron_job_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id      TEXT NOT NULL REFERENCES cron_jobs(id) ON DELETE CASCADE,
    started_at  INTEGER NOT NULL,
    finished_at INTEGER,
    status      TEXT NOT NULL DEFAULT 'running',
    result_text TEXT NOT NULL DEFAULT '',
    error_text  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_cron_history_job ON cron_job_history(job_id, started_at);
