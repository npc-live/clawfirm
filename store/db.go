// Package store provides SQLite-backed persistence for pi-go.
// Uses modernc.org/sqlite (pure Go, no CGO required).
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps a *sql.DB and exposes the store layer.
type DB struct {
	sql *sql.DB
}

// Open opens (or creates) the SQLite database at path.
// If path is empty it defaults to ~/.pi-go/data.db.
// Runs all pending migrations automatically.
func Open(path string) (*DB, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("store: home dir: %w", err)
		}
		path = filepath.Join(home, ".pi-go", "data.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("store: mkdir: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite WAL supports one writer

	d := &DB{sql: db}
	if err := d.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error { return d.sql.Close() }

// SQL returns the underlying *sql.DB for use by packages that need raw access
// (e.g. the memory index manager).
func (d *DB) SQL() *sql.DB { return d.sql }

// migrate runs SQL migration files embedded under migrations/*.sql in order.
func (d *DB) migrate() error {
	// Ensure the schema_migrations table exists.
	_, err := d.sql.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at INTEGER NOT NULL DEFAULT (unixepoch())
	)`)
	if err != nil {
		return fmt.Errorf("store: create migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("store: read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		version := strings.TrimSuffix(name, ".sql")

		var count int
		_ = d.sql.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).Scan(&count)
		if count > 0 {
			continue // already applied
		}

		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", name, err)
		}
		if _, err := d.sql.Exec(string(data)); err != nil {
			return fmt.Errorf("store: apply migration %s: %w", name, err)
		}
		if _, err := d.sql.Exec(`INSERT INTO schema_migrations(version) VALUES(?)`, version); err != nil {
			return fmt.Errorf("store: record migration %s: %w", name, err)
		}
	}
	return nil
}
