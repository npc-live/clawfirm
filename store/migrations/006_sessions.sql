-- Session persistence: structured keys, token tracking, freshness metadata

CREATE TABLE IF NOT EXISTS sessions (
    session_key         TEXT    PRIMARY KEY,
    session_id          TEXT    NOT NULL UNIQUE,
    channel_id          TEXT    NOT NULL,
    user_id             TEXT    NOT NULL,
    created_at          INTEGER NOT NULL DEFAULT (unixepoch()*1000),
    updated_at          INTEGER NOT NULL DEFAULT (unixepoch()*1000),
    label               TEXT    NOT NULL DEFAULT '',
    display_name        TEXT    NOT NULL DEFAULT '',
    subject             TEXT    NOT NULL DEFAULT '',
    chat_type           TEXT    NOT NULL DEFAULT 'direct',
    origin_json         TEXT    NOT NULL DEFAULT '{}',
    last_channel        TEXT    NOT NULL DEFAULT '',
    last_to             TEXT    NOT NULL DEFAULT '',
    last_account_id     TEXT    NOT NULL DEFAULT '',
    last_thread_id      TEXT    NOT NULL DEFAULT '',
    input_tokens        INTEGER NOT NULL DEFAULT 0,
    output_tokens       INTEGER NOT NULL DEFAULT 0,
    total_tokens        INTEGER NOT NULL DEFAULT 0,
    cache_read          INTEGER NOT NULL DEFAULT 0,
    cache_write         INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd  REAL    NOT NULL DEFAULT 0.0,
    model               TEXT    NOT NULL DEFAULT '',
    model_provider      TEXT    NOT NULL DEFAULT '',
    compaction_count    INTEGER NOT NULL DEFAULT 0,
    ended_at            INTEGER,
    system_sent         INTEGER NOT NULL DEFAULT 0,
    last_reset_at       INTEGER,
    reset_mode          TEXT    NOT NULL DEFAULT 'never',
    reset_at_hour       INTEGER NOT NULL DEFAULT 0,
    idle_minutes        INTEGER NOT NULL DEFAULT 30
);

CREATE INDEX IF NOT EXISTS idx_sessions_channel_user
    ON sessions(channel_id, user_id, updated_at);

CREATE INDEX IF NOT EXISTS idx_sessions_session_id
    ON sessions(session_id);
