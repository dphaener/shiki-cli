package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path using atomic rename
// Returns error if write or rename fails
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Create temp file in same directory (same filesystem)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}

	// Sync to disk before rename
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	// Set permissions on temp file
	if err := os.Chmod(tmpPath, perm); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}

	// Atomic rename (POSIX guarantee)
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

// AtomicWriteString is a convenience wrapper
func AtomicWriteString(path, content string, perm os.FileMode) error {
	return AtomicWrite(path, []byte(content), perm)
}
