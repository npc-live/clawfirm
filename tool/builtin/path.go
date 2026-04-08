package builtin

import (
	"os"
	"path/filepath"
	"strings"
)

// expandHome expands a leading ~ to the user's home directory.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[1:])
		}
	}
	return p
}
