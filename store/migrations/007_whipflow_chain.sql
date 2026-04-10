CREATE TABLE IF NOT EXISTS whipflow_chain (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    call_id     TEXT NOT NULL UNIQUE,
    parent_id   TEXT,
    session_idx INTEGER DEFAULT -1,
    status      TEXT DEFAULT 'running',
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_whipflow_chain_lookup
    ON whipflow_chain(channel_id, user_id);
