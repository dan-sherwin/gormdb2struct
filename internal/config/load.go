package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const CurrentConfigVersion = 1

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	configVersion, hasVersion, err := detectConfigVersion(data)
	if err != nil {
		return Config{}, fmt.Errorf("parse TOML config %s: %w", path, err)
	}

	if !hasVersion {
		return loadLegacy(data, path)
	}

	switch configVersion {
	case CurrentConfigVersion:
		return loadVersioned(data, path)
	default:
		return Config{}, fmt.Errorf("validate config %s: unsupported ConfigVersion %d", path, configVersion)
	}
}

func detectConfigVersion(data []byte) (int, bool, error) {
	raw := map[string]any{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return 0, false, err
	}

	value, exists := raw["ConfigVersion"]
	if !exists {
		return 0, false, nil
	}

	switch typed := value.(type) {
	case int64:
		if typed <= 0 {
			return 0, false, fmt.Errorf("ConfigVersion must be greater than zero")
		}
		return int(typed), true, nil
	default:
		return 0, false, fmt.Errorf("ConfigVersion must be an integer")
	}
}
