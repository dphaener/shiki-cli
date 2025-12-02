package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Load applies priority: CLI flags > env vars > file > defaults
func Load(configPath string, cliOverrides map[string]interface{}) (*Config, error) {
	// Start with defaults
	cfg := Default()

	// Apply config file if exists
	if configPath == "" {
		configPath = filepath.Join(GetConfigDir(), "config.json")
	}
	if data, err := os.ReadFile(configPath); err == nil { //nolint:gosec // G304: Reading from validated path
		_ = json.Unmarshal(data, cfg)
	}

	// Apply environment variables
	applyEnvOverrides(cfg)

	// Apply CLI flag overrides (highest priority)
	applyCLIOverrides(cfg, cliOverrides)

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if val := os.Getenv("SHIKI_CLI_WORKSPACE"); val != "" {
		cfg.WorkspaceDir = val
	}
	if val := os.Getenv("SHIKI_CLI_NO_COLOR"); val == "true" || val == "1" {
		cfg.UIColor = false
	}
	if val := os.Getenv("SHIKI_CLI_MODEL"); val != "" {
		cfg.DefaultModel = val
	}
	if val := os.Getenv("SHIKI_CLI_TIMEOUT"); val != "" {
		if timeout, err := strconv.Atoi(val); err == nil {
			cfg.TurnTimeoutSec = timeout
		}
	}
	if val := os.Getenv("SHIKI_CLI_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}
	if val := os.Getenv("SHIKI_CLI_LOG_FORMAT"); val != "" {
		cfg.LogFormat = val
	}
}

func applyCLIOverrides(cfg *Config, overrides map[string]interface{}) {
	if val, ok := overrides["workspace"].(string); ok && val != "" {
		cfg.WorkspaceDir = val
	}
	if val, ok := overrides["model"].(string); ok && val != "" {
		cfg.DefaultModel = val
	}
	if val, ok := overrides["no-color"].(bool); ok && val {
		cfg.UIColor = false
	}
	if val, ok := overrides["timeout"].(int); ok && val > 0 {
		cfg.TurnTimeoutSec = val
	}
	if val, ok := overrides["log-level"].(string); ok && val != "" {
		cfg.LogLevel = val
	}
}
