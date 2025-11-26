package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/darinhaener/collab/internal/config"
)

var (
	counterMu sync.Mutex
)

// GetNextFeatureNumber returns the next available feature number
// and increments the counter atomically
func GetNextFeatureNumber() (int, error) {
	counterMu.Lock()
	defer counterMu.Unlock()

	counterPath := filepath.Join(config.GetDataDir(), "specs", ".counter")

	// Read current counter
	data, err := os.ReadFile(counterPath) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		if os.IsNotExist(err) {
			// First feature
			if err := saveCounter(counterPath, 1); err != nil {
				return 0, err
			}
			return 1, nil
		}
		return 0, fmt.Errorf("read counter: %w", err)
	}

	current, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse counter: %w", err)
	}

	next := current + 1
	if err := saveCounter(counterPath, next); err != nil {
		return 0, err
	}

	return current, nil
}

// GetCurrentFeatureNumber returns the current counter value without incrementing
func GetCurrentFeatureNumber() (int, error) {
	counterPath := filepath.Join(config.GetDataDir(), "specs", ".counter")

	data, err := os.ReadFile(counterPath) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read counter: %w", err)
	}

	current, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse counter: %w", err)
	}

	return current, nil
}

// saveCounter writes the counter value atomically
func saveCounter(path string, value int) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create specs directory: %w", err)
	}

	content := fmt.Sprintf("%d\n", value)
	return AtomicWriteString(path, content, 0o644)
}
