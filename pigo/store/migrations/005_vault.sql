-- Vault: encrypted-at-rest KV store for secrets injected as env vars.
CREATE TABLE IF NOT EXISTS vault (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
