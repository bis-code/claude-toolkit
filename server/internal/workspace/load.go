package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadConfig reads a .claude-workspace.json file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read workspace config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse workspace config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig writes a Config to a .claude-workspace.json file.
func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal workspace config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadOrDetect loads from a config file if it exists, otherwise auto-detects.
// When a config file exists, it merges user config over auto-detected results.
func LoadOrDetect(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ".claude-workspace.json")

	detected, err := Detect(dir)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); err == nil {
		loaded, err := LoadConfig(configPath)
		if err != nil {
			return nil, err
		}
		return merge(detected, loaded), nil
	}

	return detected, nil
}

// merge combines auto-detected config with user-provided config.
// User config takes precedence for fields it specifies.
func merge(detected, userConfig *Config) *Config {
	result := *userConfig

	if len(result.Repos) == 0 {
		result.Repos = detected.Repos
	}

	if result.Name == "" {
		result.Name = detected.Name
	}

	if len(result.Shared) == 0 {
		result.Shared = detected.Shared
	}

	return &result
}
