-- Initial schema

CREATE TABLE IF NOT EXISTS messages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  TEXT    NOT NULL,
    user_id     TEXT    NOT NULL,
    role        TEXT    NOT NULL,   -- "user" | "assistant" | "tool"
    content     TEXT    NOT NULL,   -- JSON-encoded []ContentBlock
    created_at  INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_user ON messages(channel_id, user_id, created_at);

CREATE TABLE IF NOT EXISTS kv (
    key         TEXT    PRIMARY KEY,
    value       TEXT    NOT NULL,   -- JSON value
    updated_at  INTEGER NOT NULL DEFAULT (unixepoch())
);
