package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns XDG_CONFIG_HOME/collab-cli or equivalent
func GetConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "collab-cli")
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		return filepath.Join(home, ".config", "collab-cli")
	}
	// Windows fallback
	return filepath.Join(home, "AppData", "Local", "collab-cli")
}

// GetDataDir returns XDG_DATA_HOME/collab-cli or equivalent
func GetDataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "collab-cli")
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		return filepath.Join(home, ".local", "share", "collab-cli")
	}
	// Windows fallback
	return filepath.Join(home, "AppData", "Local", "collab-cli", "data")
}
