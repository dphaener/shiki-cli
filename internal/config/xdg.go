package config

import (
	"fmt"
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

// GetStateDir returns XDG_STATE_HOME/collab-cli or equivalent
func GetStateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "collab-cli")
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		return filepath.Join(home, ".local", "state", "collab-cli")
	}
	// Windows fallback - use same as data dir
	return filepath.Join(home, "AppData", "Local", "collab-cli", "state")
}

// EnsureLogDirectories creates the necessary directories for logging with proper permissions
func EnsureLogDirectories() error {
	dirs := []string{
		GetStateDir(),
		GetDataDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}
	}
	return nil
}

// GetDebugLogPath returns the path to the debug log file, creating directories if needed
func GetDebugLogPath() (string, error) {
	stateDir := GetStateDir()

	// Try to create the directory first
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		// If XDG directory creation fails, fall back to current directory
		return filepath.Join(".", "collab-debug.log"), nil
	}

	return filepath.Join(stateDir, "debug.log"), nil
}

// GetAssistantLogPath returns the path to the assistant log file, creating directories if needed
func GetAssistantLogPath() (string, error) {
	dataDir := GetDataDir()

	// Try to create the directory first
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// If XDG directory creation fails, fall back to current directory
		return filepath.Join(".", "assistant.log"), nil
	}

	return filepath.Join(dataDir, "assistant.log"), nil
}

// MigrateDebugLog moves the existing collab-debug.log to the XDG state directory
func MigrateDebugLog() error {
	oldPath := filepath.Join(".", "collab-debug.log")

	// Check if old log file exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		// No old file to migrate
		return nil
	}

	// Get new debug log path
	newPath, err := GetDebugLogPath()
	if err != nil {
		return fmt.Errorf("failed to get debug log path: %w", err)
	}

	// If new path is same as old path (fallback case), no migration needed
	if oldPath == newPath {
		return nil
	}

	// Check if destination already exists
	if _, err := os.Stat(newPath); err == nil {
		// Destination exists, don't overwrite
		return nil
	}

	// Perform the migration (move, not copy)
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to migrate debug log from %s to %s: %w", oldPath, newPath, err)
	}

	return nil
}
