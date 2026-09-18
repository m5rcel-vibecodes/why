package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	RulesDir string `yaml:"rules_dir"`
	Color    string `yaml:"color"`
	JSON     bool   `yaml:"json"`
}

// DefaultConfig returns safe defaults.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	rulesDir := ""
	if home != "" {
		rulesDir = filepath.Join(home, ".config", "why", "rules.d")
	}

	return &Config{
		RulesDir: rulesDir,
		Color:    "auto",
		JSON:     false,
	}
}

// Load reads configuration from the given path, or fallback to default locations.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			configPath = filepath.Join(home, ".config", "why", "config.yaml")
		}
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			_ = yaml.Unmarshal(data, cfg)
		}
	}

	// Environment variable overrides
	if dir := os.Getenv("WHY_RULES_DIR"); dir != "" {
		cfg.RulesDir = dir
	}
	if col := os.Getenv("WHY_COLOR"); col != "" {
		cfg.Color = col
	}
	if os.Getenv("WHY_JSON") == "1" || os.Getenv("WHY_JSON") == "true" {
		cfg.JSON = true
	}

	return cfg, nil
}
