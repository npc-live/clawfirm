//go:build !darwin

package keychain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LinuxKeychain stores the key in a file under ~/.clawfirm/.key
// with 0600 permissions. This is a fallback for non-macOS systems.
type LinuxKeychain struct {
	dir string
}

func New() Keychain {
	home, _ := os.UserHomeDir()
	return &LinuxKeychain{dir: filepath.Join(home, ".clawfirm")}
}

func (k *LinuxKeychain) keyPath(service string) string {
	return filepath.Join(k.dir, "."+service+".key")
}

func (k *LinuxKeychain) GetKey(service string) ([]byte, error) {
	data, err := os.ReadFile(k.keyPath(service))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("key not found for service %q", service)
		}
		return nil, err
	}
	hexStr := strings.TrimSpace(string(data))
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, errors.New("keyfile: stored value is not valid hex")
	}
	return decoded, nil
}

func (k *LinuxKeychain) SetKey(service string, key []byte) error {
	if err := os.MkdirAll(k.dir, 0700); err != nil {
		return err
	}
	hexStr := hex.EncodeToString(key)
	return os.WriteFile(k.keyPath(service), []byte(hexStr), 0600)
}
