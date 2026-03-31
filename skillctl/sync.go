package skillctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClientDirs maps client names to their skill directories.
var ClientDirs = map[string]string{
	"claude-code": "~/.claude/skills",
	"cursor":      "~/.cursor/skills",
	"clawfirm":    "~/.clawfirm/skills",
	"opencode":    "~/.config/opencode/skills",
}

// SyncOptions configures a Sync operation.
type SyncOptions struct {
	RegistryPath string // defaults to DefaultRegistryPath()
	SkillsDir    string // defaults to DefaultSkillsDir()
	Client       string // if set, only sync this client
	DryRun       bool
}

// SymlinkAction describes a single symlink create/remove action.
type SymlinkAction struct {
	Client   string
	Skill    string
	LinkPath string
	Target   string
}

// SyncResult summarizes what Sync did.
type SyncResult struct {
	Created []SymlinkAction
	Removed []SymlinkAction
	Skipped []SymlinkAction
}

// Sync reads the registry and creates/removes symlinks for each enabled client × enabled skill.
func Sync(opts SyncOptions) (*SyncResult, error) {
	regPath := opts.RegistryPath
	if regPath == "" {
		regPath = DefaultRegistryPath()
	}
	skillsDir := opts.SkillsDir
	if skillsDir == "" {
		skillsDir = DefaultSkillsDir()
	}

	reg, err := LoadRegistry(regPath)
	if err != nil {
		return nil, fmt.Errorf("skillctl: load registry: %w", err)
	}

	result := &SyncResult{}

	for clientName, clientDir := range ClientDirs {
		if opts.Client != "" && opts.Client != clientName {
			continue
		}
		// Check if client is enabled in registry (default: enabled if not listed).
		if ce, listed := reg.Clients[clientName]; listed && !ce.Enabled {
			continue
		}

		expandedDir := expandHome(clientDir)

		for skillName, se := range reg.Skills {
			if !se.Enabled {
				continue
			}
			// If the skill has a specific client list, check it.
			if len(se.Clients) > 0 && !containsString(se.Clients, clientName) {
				continue
			}

			target := filepath.Join(skillsDir, skillName)
			linkPath := filepath.Join(expandedDir, skillName)

			action := SymlinkAction{
				Client:   clientName,
				Skill:    skillName,
				LinkPath: linkPath,
				Target:   target,
			}

			// Check if target skill exists.
			if _, err := os.Stat(target); os.IsNotExist(err) {
				action.Target = target
				result.Skipped = append(result.Skipped, action)
				continue
			}

			// Check if symlink already exists and points to the right place.
			if existing, err := os.Readlink(linkPath); err == nil && existing == target {
				result.Skipped = append(result.Skipped, action)
				continue
			}

			if opts.DryRun {
				result.Created = append(result.Created, action)
				continue
			}

			if err := os.MkdirAll(expandedDir, 0o755); err != nil {
				return nil, fmt.Errorf("skillctl: mkdir %s: %w", expandedDir, err)
			}
			// Remove stale entry (file, broken symlink, etc.).
			_ = os.Remove(linkPath)

			if err := os.Symlink(target, linkPath); err != nil {
				return nil, fmt.Errorf("skillctl: symlink %s → %s: %w", linkPath, target, err)
			}
			result.Created = append(result.Created, action)
		}

		// Remove symlinks for disabled/removed skills.
		entries, err := os.ReadDir(expandedDir)
		if err != nil {
			continue // directory may not exist
		}
		for _, e := range entries {
			linkPath := filepath.Join(expandedDir, e.Name())
			target, err := os.Readlink(linkPath)
			if err != nil {
				continue // not a symlink
			}
			// Only manage symlinks pointing into the skillctl skills directory.
			if !strings.HasPrefix(target, skillsDir+string(os.PathSeparator)) && target != skillsDir {
				continue
			}
			skillName := e.Name()
			se, exists := reg.Skills[skillName]
			shouldExist := exists && se.Enabled
			if shouldExist && (len(se.Clients) == 0 || containsString(se.Clients, clientName)) {
				continue
			}

			action := SymlinkAction{
				Client:   clientName,
				Skill:    skillName,
				LinkPath: linkPath,
				Target:   target,
			}
			if opts.DryRun {
				result.Removed = append(result.Removed, action)
				continue
			}
			if err := os.Remove(linkPath); err == nil {
				result.Removed = append(result.Removed, action)
			}
		}
	}

	return result, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
