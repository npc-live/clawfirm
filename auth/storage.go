package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// APIKeyCredentials stores an API key with a timestamp.
type APIKeyCredentials struct {
	Key       string    `json:"key"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OAuthCredentials stores OAuth token data for a provider.
type OAuthCredentials struct {
	Refresh   string         `json:"refresh"`
	Access    string         `json:"access"`
	ExpiresAt time.Time      `json:"expiresAt"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// StoredCredentials is the on-disk credential file format.
type StoredCredentials struct {
	APIKeys map[string]APIKeyCredentials `json:"apiKeys"`
	OAuth   map[string]OAuthCredentials  `json:"oauth"`
}

// AuthStorage persists and retrieves credentials.
// Runtime keys (set via SetRuntimeKey) take priority and are never written to disk.
type AuthStorage struct {
	path        string
	mu          sync.RWMutex
	runtimeKeys map[string]string
	stored      StoredCredentials
}

// NewAuthStorage creates an AuthStorage backed by the given file path.
// If path is empty, defaults to ~/.pi-go/auth.json.
func NewAuthStorage(path string) *AuthStorage {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".pi-go", "auth.json")
	}
	return &AuthStorage{
		path:        path,
		runtimeKeys: make(map[string]string),
		stored: StoredCredentials{
			APIKeys: make(map[string]APIKeyCredentials),
			OAuth:   make(map[string]OAuthCredentials),
		},
	}
}

// SetRuntimeKey sets an in-memory-only API key for the given provider.
// Runtime keys are never persisted and take highest priority.
func (s *AuthStorage) SetRuntimeKey(provider, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeKeys[provider] = key
}

// GetAPIKey returns the API key for the provider.
// Priority: runtime key > stored key.
func (s *AuthStorage) GetAPIKey(provider string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if k, ok := s.runtimeKeys[provider]; ok && k != "" {
		return k, true
	}
	if c, ok := s.stored.APIKeys[provider]; ok && c.Key != "" {
		return c.Key, true
	}
	return "", false
}

// SetAPIKey saves an API key for the provider to disk.
func (s *AuthStorage) SetAPIKey(provider, key string) error {
	s.mu.Lock()
	s.stored.APIKeys[provider] = APIKeyCredentials{Key: key, UpdatedAt: time.Now()}
	s.mu.Unlock()
	return s.Save()
}

// GetOAuth returns OAuth credentials for the provider.
func (s *AuthStorage) GetOAuth(provider string) (*OAuthCredentials, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.stored.OAuth[provider]
	if !ok {
		return nil, false
	}
	cp := c
	return &cp, true
}

// SetOAuth saves OAuth credentials for the provider to disk.
func (s *AuthStorage) SetOAuth(provider string, creds OAuthCredentials) error {
	s.mu.Lock()
	s.stored.OAuth[provider] = creds
	s.mu.Unlock()
	return s.Save()
}

// Load reads stored credentials from disk.
// If the file does not exist, Load succeeds and leaves storage empty.
func (s *AuthStorage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var stored StoredCredentials
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	if stored.APIKeys == nil {
		stored.APIKeys = make(map[string]APIKeyCredentials)
	}
	if stored.OAuth == nil {
		stored.OAuth = make(map[string]OAuthCredentials)
	}
	s.stored = stored
	return nil
}

// Save writes stored credentials to disk atomically.
func (s *AuthStorage) Save() error {
	s.mu.RLock()
	stored := s.stored
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file then rename for atomicity
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
