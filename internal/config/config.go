package config

import "path/filepath"

// Config holds all configuration
type Config struct {
	WorkspaceDir      string  `json:"workspace_dir"`
	DefaultModel      string  `json:"default_model"`
	TurnTimeoutSec    int     `json:"turn_timeout_seconds"`
	CostLimitSession  float64 `json:"cost_limit_session"`
	CostLimitPerAgent float64 `json:"cost_limit_per_agent"`
	UIColor           bool    `json:"ui_color"`
	UIRefreshRateFPS  int     `json:"ui_refresh_rate_fps"`
	LogLevel          string  `json:"log_level"`
	LogFormat         string  `json:"log_format"`
}

// Default returns config with sensible defaults
func Default() *Config {
	return &Config{
		WorkspaceDir:      filepath.Join(GetDataDir(), "sessions"),
		DefaultModel:      "claude-sonnet-4",
		TurnTimeoutSec:    300,
		CostLimitSession:  10.0,
		CostLimitPerAgent: 5.0,
		UIColor:           true,
		UIRefreshRateFPS:  10,
		LogLevel:          "info",
		LogFormat:         "json",
	}
}
