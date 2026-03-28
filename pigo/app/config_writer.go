package app

import (
	"os"

	"github.com/ai-gateway/pi-go/config"
	"gopkg.in/yaml.v3"
)

// writeConfigYAML serializes cfg to YAML and writes it to path atomically.
func writeConfigYAML(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
