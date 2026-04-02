package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// sessionDir returns the directory for saved sessions.
func sessionDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawfirm", "sessions")
}

func sessionPath(platform string) string {
	return filepath.Join(sessionDir(), platform+".json")
}

// SaveSession persists cookies to disk for the given platform.
func SaveSession(platform string, cookies []Cookie) error {
	dir := sessionDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath(platform), data, 0o644)
}

// LoadSession reads previously saved cookies for a platform.
// Returns nil if no session file exists.
func LoadSession(platform string) ([]Cookie, error) {
	data, err := os.ReadFile(sessionPath(platform))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cookies []Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, err
	}
	return cookies, nil
}

// HasSession checks whether a session file exists for the platform.
func HasSession(platform string) bool {
	_, err := os.Stat(sessionPath(platform))
	return err == nil
}

// CaptureSession reads all cookies from the browser and saves them.
func CaptureSession(client *CDPClient, platform string) ([]Cookie, error) {
	cookies, err := client.GetAllCookies()
	if err != nil {
		return nil, fmt.Errorf("capture session: %w", err)
	}
	if err := SaveSession(platform, cookies); err != nil {
		return nil, fmt.Errorf("capture session: %w", err)
	}
	return cookies, nil
}

// RestoreSession loads saved cookies and sets them in the browser.
func RestoreSession(client *CDPClient, platform string) (bool, error) {
	cookies, err := LoadSession(platform)
	if err != nil {
		return false, err
	}
	if cookies == nil {
		return false, nil
	}
	if err := client.SetCookies(cookies); err != nil {
		return false, err
	}
	return true, nil
}
