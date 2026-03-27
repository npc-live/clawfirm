-- Memory index: Markdown files + chunks + FTS5 + embedding cache

CREATE TABLE IF NOT EXISTS memory_files (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT    NOT NULL UNIQUE,
    hash        TEXT    NOT NULL,           -- SHA-256 hex of file content
    modified_at INTEGER NOT NULL,           -- file mtime as unix seconds
    indexed_at  INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS memory_chunks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id     INTEGER NOT NULL REFERENCES memory_files(id) ON DELETE CASCADE,
    chunk_idx   INTEGER NOT NULL,           -- 0-based position within file
    content     TEXT    NOT NULL,
    start_line  INTEGER NOT NULL,           -- 1-based inclusive
    end_line    INTEGER NOT NULL,           -- 1-based inclusive
    embedding   BLOB,                       -- float32 LE, NULL until embedded
    created_at  INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_memory_chunks_file ON memory_chunks(file_id);

-- FTS5 full-text index (content table, synced via triggers below)
CREATE VIRTUAL TABLE IF NOT EXISTS memory_chunks_fts USING fts5(
    content,
    content='memory_chunks',
    content_rowid='id',
    tokenize='unicode61'
);

CREATE TRIGGER IF NOT EXISTS memory_chunks_ai
    AFTER INSERT ON memory_chunks BEGIN
        INSERT INTO memory_chunks_fts(rowid, content) VALUES (new.id, new.content);
    END;

CREATE TRIGGER IF NOT EXISTS memory_chunks_ad
    AFTER DELETE ON memory_chunks BEGIN
        INSERT INTO memory_chunks_fts(memory_chunks_fts, rowid, content)
        VALUES ('delete', old.id, old.content);
    END;

CREATE TRIGGER IF NOT EXISTS memory_chunks_au
    AFTER UPDATE OF content ON memory_chunks BEGIN
        INSERT INTO memory_chunks_fts(memory_chunks_fts, rowid, content)
        VALUES ('delete', old.id, old.content);
        INSERT INTO memory_chunks_fts(rowid, content) VALUES (new.id, new.content);
    END;

-- Embedding result cache keyed by SHA-256(text)+provider+model
CREATE TABLE IF NOT EXISTS memory_embedding_cache (
    text_hash   TEXT    NOT NULL,
    provider    TEXT    NOT NULL,
    model       TEXT    NOT NULL,
    embedding   BLOB    NOT NULL,           -- float32 LE
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (text_hash, provider, model)
);
