package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.DefaultModel != "claude-sonnet-4" {
		t.Errorf("DefaultModel = %s, want claude-sonnet-4", cfg.DefaultModel)
	}
	if cfg.TurnTimeoutSec != 300 {
		t.Errorf("TurnTimeoutSec = %d, want 300", cfg.TurnTimeoutSec)
	}
	if !cfg.UIColor {
		t.Error("UIColor should be true by default")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %s, want info", cfg.LogLevel)
	}
}

func TestEnvOverrides(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("COLLAB_CLI_MODEL", "claude-opus-4")
	_ = os.Setenv("COLLAB_CLI_NO_COLOR", "true")
	defer func() {
		_ = os.Unsetenv("COLLAB_CLI_MODEL")
		_ = os.Unsetenv("COLLAB_CLI_NO_COLOR")
	}()

	cfg, err := Load("", nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DefaultModel != "claude-opus-4" {
		t.Errorf("DefaultModel = %s, want claude-opus-4 (env override)", cfg.DefaultModel)
	}
	if cfg.UIColor {
		t.Error("UIColor should be false when COLLAB_CLI_NO_COLOR=true")
	}
}

func TestCLIOverrides(t *testing.T) {
	overrides := map[string]interface{}{
		"model":    "custom-model",
		"no-color": true,
		"timeout":  600,
	}

	cfg, err := Load("", overrides)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DefaultModel != "custom-model" {
		t.Errorf("DefaultModel = %s, want custom-model (CLI override)", cfg.DefaultModel)
	}
	if cfg.UIColor {
		t.Error("UIColor should be false when no-color=true")
	}
	if cfg.TurnTimeoutSec != 600 {
		t.Errorf("TurnTimeoutSec = %d, want 600 (CLI override)", cfg.TurnTimeoutSec)
	}
}

func TestPriorityOrder(t *testing.T) {
	// Set env var
	_ = os.Setenv("COLLAB_CLI_MODEL", "env-model")
	defer func() { _ = os.Unsetenv("COLLAB_CLI_MODEL") }()

	// CLI override should win
	overrides := map[string]interface{}{
		"model": "cli-model",
	}

	cfg, err := Load("", overrides)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DefaultModel != "cli-model" {
		t.Errorf("DefaultModel = %s, want cli-model (CLI should override env)", cfg.DefaultModel)
	}
}
