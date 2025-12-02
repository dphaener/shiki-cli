package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/dphaener/shiki-cli/internal/config"
)

var (
	counterMu          sync.Mutex
	featureNumberRegex = regexp.MustCompile(`^(\d{3})-`)
)

// GetNextFeatureNumber returns the next available feature number
// and increments the counter atomically. It scans existing spec directories
// to ensure the number is always higher than any existing specs.
func GetNextFeatureNumber() (int, error) {
	counterMu.Lock()
	defer counterMu.Unlock()

	counterPath := filepath.Join(config.GetDataDir(), "specs", ".counter")

	// Get the highest existing feature number from directories
	maxExisting := getMaxExistingFeatureNumber()

	// Read current counter value
	counterValue := 0
	data, err := os.ReadFile(counterPath) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		if !os.IsNotExist(err) {
			return 0, fmt.Errorf("read counter: %w", err)
		}
		// Counter file doesn't exist, that's fine
	} else {
		counterValue, err = strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return 0, fmt.Errorf("parse counter: %w", err)
		}
	}

	// Use the max of counter value and existing specs
	base := counterValue
	if maxExisting > base {
		base = maxExisting
	}

	// Next available number
	next := base + 1

	// Save counter for next call
	if err := saveCounter(counterPath, next); err != nil {
		return 0, err
	}

	return next, nil
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

// getMaxExistingFeatureNumber scans spec directories and returns the highest
// feature number found in directory names (e.g., "005" from "005-feature-name")
func getMaxExistingFeatureNumber() int {
	specsDir := filepath.Join(config.GetDataDir(), "specs")

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return 0
	}

	maxNum := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := featureNumberRegex.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}

		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		if num > maxNum {
			maxNum = num
		}
	}

	return maxNum
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
