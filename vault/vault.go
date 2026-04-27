package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ai-gateway/clawfirm/vault/crypto"
	"github.com/ai-gateway/clawfirm/vault/keychain"
	"github.com/ai-gateway/clawfirm/vault/store"
)

const (
	ServiceName = "clawfirm"
	DBFileName  = "vault.db"
	DirPerm     = 0700
)

// DefaultDir returns the clawfirm data directory, honouring CLAWFIRM_DATA_DIR.
func DefaultDir() string {
	if dir := os.Getenv("CLAWFIRM_DATA_DIR"); dir != "" {
		if len(dir) >= 2 && dir[:2] == "~/" {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, dir[2:])
		}
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ServiceName)
	}
	return filepath.Join(home, ".clawfirm")
}

// DefaultDBPath returns ~/.clawfirm/vault.db.
func DefaultDBPath() string {
	return filepath.Join(DefaultDir(), DBFileName)
}

// Vault is the high-level orchestrator that ties together the store,
// keychain, and crypto layers.
type Vault struct {
	st  store.Store
	key []byte
}

// Init creates a new vault: generates an encryption key, stores it in the
// keychain, creates the DB directory and opens the store.
func Init(dbPath string, kc keychain.Keychain) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, DirPerm); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	if err := kc.SetKey(ServiceName, key); err != nil {
		return fmt.Errorf("store key in keychain: %w", err)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	return st.Close()
}

// Open opens an existing vault.
func Open(dbPath string, kc keychain.Keychain) (*Vault, error) {
	key, err := kc.GetKey(ServiceName)
	if err != nil {
		return nil, fmt.Errorf("get encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("invalid key length in keychain")
	}

	st, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	return &Vault{st: st, key: key}, nil
}

// Set encrypts and stores a secret.
func (v *Vault) Set(name string, value []byte) error {
	enc, err := crypto.Encrypt(v.key, value)
	if err != nil {
		return fmt.Errorf("encrypt %q: %w", name, err)
	}
	return v.st.Set(name, enc)
}

// Get retrieves and decrypts a secret.
func (v *Vault) Get(name string) ([]byte, error) {
	enc, err := v.st.Get(name)
	if err != nil {
		return nil, err
	}
	plain, err := crypto.Decrypt(v.key, enc)
	if err != nil {
		return nil, fmt.Errorf("decrypt %q: %w", name, err)
	}
	return plain, nil
}

// Delete removes a secret.
func (v *Vault) Delete(name string) error {
	return v.st.Delete(name)
}

// List returns all secret names (excluding internal variant keys).
func (v *Vault) List() ([]string, error) {
	all, err := v.st.List()
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, k := range all {
		if !IsInternalKey(k) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// Env returns all secrets as KEY=value pairs suitable for environment injection.
// Internal variant keys are excluded.
func (v *Vault) Env() (map[string]string, error) {
	all, err := v.st.List()
	if err != nil {
		return nil, err
	}
	env := make(map[string]string, len(all))
	for _, k := range all {
		if IsInternalKey(k) {
			continue
		}
		val, err := v.Get(k)
		if err != nil {
			return nil, fmt.Errorf("get %q: %w", k, err)
		}
		env[k] = string(val)
	}
	return env, nil
}

// Close zeroes the in-memory key and closes the store.
func (v *Vault) Close() error {
	for i := range v.key {
		v.key[i] = 0
	}
	return v.st.Close()
}
