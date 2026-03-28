-- Migrate FTS5 index from unicode61 to trigram tokenizer.
-- trigram supports CJK (Chinese/Japanese/Korean) character search
-- whereas unicode61 only tokenises ASCII-like scripts.
-- Requires every search token to be >= 3 Unicode code points;
-- the ftsQuery helper in memory/manager.go handles this by sliding
-- a 3-rune window over short CJK strings.

DROP TABLE IF EXISTS memory_chunks_fts;

CREATE VIRTUAL TABLE memory_chunks_fts USING fts5(
    content,
    content='memory_chunks',
    content_rowid='id',
    tokenize='trigram'
);

-- Repopulate from existing chunks.
INSERT INTO memory_chunks_fts(rowid, content)
SELECT id, content FROM memory_chunks;

-- Recreate the three sync triggers.
DROP TRIGGER IF EXISTS memory_chunks_ai;
DROP TRIGGER IF EXISTS memory_chunks_ad;
DROP TRIGGER IF EXISTS memory_chunks_au;

CREATE TRIGGER memory_chunks_ai
    AFTER INSERT ON memory_chunks BEGIN
        INSERT INTO memory_chunks_fts(rowid, content) VALUES (new.id, new.content);
    END;

CREATE TRIGGER memory_chunks_ad
    AFTER DELETE ON memory_chunks BEGIN
        INSERT INTO memory_chunks_fts(memory_chunks_fts, rowid, content)
        VALUES ('delete', old.id, old.content);
    END;

CREATE TRIGGER memory_chunks_au
    AFTER UPDATE OF content ON memory_chunks BEGIN
        INSERT INTO memory_chunks_fts(memory_chunks_fts, rowid, content)
        VALUES ('delete', old.id, old.content);
        INSERT INTO memory_chunks_fts(rowid, content) VALUES (new.id, new.content);
    END;
