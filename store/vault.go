package store

import "database/sql"

// VaultEntry is a single key-value secret stored in the vault.
type VaultEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt int64  `json:"updated_at"`
}

// VaultStore provides CRUD access to the vault table.
type VaultStore struct{ db *sql.DB }

func (d *DB) Vault() *VaultStore { return &VaultStore{db: d.sql} }

// List returns all vault entries ordered by key.
func (v *VaultStore) List() ([]VaultEntry, error) {
	rows, err := v.db.Query(`SELECT key, value, updated_at FROM vault ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VaultEntry
	for rows.Next() {
		var e VaultEntry
		if err := rows.Scan(&e.Key, &e.Value, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Set inserts or replaces a vault entry.
func (v *VaultStore) Set(key, value string) error {
	_, err := v.db.Exec(
		`INSERT INTO vault(key,value,updated_at) VALUES(?,?,unixepoch())
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=unixepoch()`,
		key, value,
	)
	return err
}

// Delete removes a vault entry by key.
func (v *VaultStore) Delete(key string) error {
	_, err := v.db.Exec(`DELETE FROM vault WHERE key=?`, key)
	return err
}

// Map returns all entries as a key→value map (for env injection).
func (v *VaultStore) Map() (map[string]string, error) {
	entries, err := v.List()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.Value
	}
	return m, nil
}
