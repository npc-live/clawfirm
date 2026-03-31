//go:build darwin

package keychain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// DarwinKeychain uses the macOS `security` CLI subprocess.
type DarwinKeychain struct{}

func New() Keychain {
	return &DarwinKeychain{}
}

func (k *DarwinKeychain) GetKey(service string) ([]byte, error) {
	out, err := exec.Command(
		"security", "find-generic-password",
		"-s", service,
		"-w",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("keychain get: %w", err)
	}
	hexStr := strings.TrimSpace(string(out))
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, errors.New("keychain: stored value is not valid hex")
	}
	return decoded, nil
}

func (k *DarwinKeychain) SetKey(service string, key []byte) error {
	hexStr := hex.EncodeToString(key)
	cmd := exec.Command(
		"security", "add-generic-password",
		"-s", service,
		"-a", service,
		"-w", hexStr,
		"-U", // update if exists
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain set: %w: %s", err, out)
	}
	return nil
}
