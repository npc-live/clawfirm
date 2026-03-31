// Package skillctl provides skill management: registry YAML handling,
// syncing skills to client directories, and a remote registry API client.
package skillctl

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Registry represents the ~/.skillctl/registry.yaml file.
type Registry struct {
	Version int                    `yaml:"version"`
	Skills  map[string]SkillEntry  `yaml:"skills"`
	Clients map[string]ClientEntry `yaml:"clients"`
}

// SkillEntry describes one skill in the registry.
type SkillEntry struct {
	Enabled bool     `yaml:"enabled"`
	Clients []string `yaml:"clients,omitempty"`
}

// ClientEntry describes one client in the registry.
type ClientEntry struct {
	Enabled bool `yaml:"enabled"`
}

// DefaultRegistryPath returns ~/.skillctl/registry.yaml.
func DefaultRegistryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".skillctl", "registry.yaml")
}

// DefaultSkillsDir returns ~/.skillctl/skills/.
func DefaultSkillsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".skillctl", "skills")
}

// LoadRegistry reads a YAML registry file. Returns an empty registry if the file is missing.
func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{
			Version: 1,
			Skills:  make(map[string]SkillEntry),
			Clients: make(map[string]ClientEntry),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	if reg.Skills == nil {
		reg.Skills = make(map[string]SkillEntry)
	}
	if reg.Clients == nil {
		reg.Clients = make(map[string]ClientEntry)
	}
	return &reg, nil
}

// SaveRegistry writes the registry to a YAML file, creating directories as needed.
func SaveRegistry(path string, reg *Registry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(reg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
