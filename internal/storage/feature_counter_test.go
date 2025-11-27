package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMaxExistingFeatureNumber(t *testing.T) {
	// Save original data dir and restore after test
	originalDataDir := os.Getenv("XDG_DATA_HOME")
	defer func() {
		if originalDataDir == "" {
			os.Unsetenv("XDG_DATA_HOME")
		} else {
			os.Setenv("XDG_DATA_HOME", originalDataDir)
		}
	}()

	tests := []struct {
		name        string
		directories []string
		expected    int
	}{
		{
			name:        "no directories",
			directories: []string{},
			expected:    0,
		},
		{
			name:        "single spec directory",
			directories: []string{"001-feature-one"},
			expected:    1,
		},
		{
			name:        "multiple spec directories",
			directories: []string{"001-feature-one", "002-feature-two", "003-feature-three"},
			expected:    3,
		},
		{
			name:        "non-sequential numbers",
			directories: []string{"001-first", "005-fifth", "003-third"},
			expected:    5,
		},
		{
			name:        "mixed valid and invalid directories",
			directories: []string{"001-valid", "invalid-dir", "002-also-valid", ".hidden", "readme.md"},
			expected:    2,
		},
		{
			name:        "high numbers",
			directories: []string{"099-almost-max", "100-one-hundred"},
			expected:    100,
		},
		{
			name:        "directories without number prefix",
			directories: []string{"feature-one", "another-feature"},
			expected:    0,
		},
		{
			name:        "counter file should be ignored",
			directories: []string{"001-feature", ".counter"},
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			os.Setenv("XDG_DATA_HOME", tmpDir)

			specsDir := filepath.Join(tmpDir, "collab-cli", "specs")
			require.NoError(t, os.MkdirAll(specsDir, 0o755))

			// Create test directories
			for _, dir := range tt.directories {
				path := filepath.Join(specsDir, dir)
				if filepath.Ext(dir) != "" || dir == ".counter" {
					// It's a file, create it
					require.NoError(t, os.WriteFile(path, []byte("test"), 0o644))
				} else {
					// It's a directory
					require.NoError(t, os.MkdirAll(path, 0o755))
				}
			}

			result := getMaxExistingFeatureNumber()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNextFeatureNumber_SyncsWithExistingSpecs(t *testing.T) {
	// Save original data dir and restore after test
	originalDataDir := os.Getenv("XDG_DATA_HOME")
	defer func() {
		if originalDataDir == "" {
			os.Unsetenv("XDG_DATA_HOME")
		} else {
			os.Setenv("XDG_DATA_HOME", originalDataDir)
		}
	}()

	tests := []struct {
		name           string
		existingSpecs  []string
		counterValue   string // empty means no counter file
		expectedNumber int
	}{
		{
			name:           "no specs, no counter",
			existingSpecs:  []string{},
			counterValue:   "",
			expectedNumber: 1,
		},
		{
			name:           "counter behind existing specs",
			existingSpecs:  []string{"001-first", "002-second", "005-fifth"},
			counterValue:   "3",
			expectedNumber: 6, // max(3, 5) + 1 = 6
		},
		{
			name:           "counter ahead of existing specs",
			existingSpecs:  []string{"001-first", "002-second"},
			counterValue:   "10",
			expectedNumber: 11, // max(10, 2) + 1 = 11
		},
		{
			name:           "no counter, existing specs",
			existingSpecs:  []string{"001-first", "002-second", "003-third"},
			counterValue:   "",
			expectedNumber: 4, // max(0, 3) + 1 = 4
		},
		{
			name:           "counter and specs equal",
			existingSpecs:  []string{"001-first", "002-second", "003-third"},
			counterValue:   "3",
			expectedNumber: 4, // max(3, 3) + 1 = 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			os.Setenv("XDG_DATA_HOME", tmpDir)

			specsDir := filepath.Join(tmpDir, "collab-cli", "specs")
			require.NoError(t, os.MkdirAll(specsDir, 0o755))

			// Create existing spec directories
			for _, dir := range tt.existingSpecs {
				require.NoError(t, os.MkdirAll(filepath.Join(specsDir, dir), 0o755))
			}

			// Create counter file if specified
			if tt.counterValue != "" {
				counterPath := filepath.Join(specsDir, ".counter")
				require.NoError(t, os.WriteFile(counterPath, []byte(tt.counterValue+"\n"), 0o644))
			}

			result, err := GetNextFeatureNumber()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedNumber, result)

			// Verify counter was updated
			counterPath := filepath.Join(specsDir, ".counter")
			data, err := os.ReadFile(counterPath)
			require.NoError(t, err)
			assert.Contains(t, string(data), "")
		})
	}
}

func TestGetNextFeatureNumber_ConsecutiveCalls(t *testing.T) {
	// Save original data dir and restore after test
	originalDataDir := os.Getenv("XDG_DATA_HOME")
	defer func() {
		if originalDataDir == "" {
			os.Unsetenv("XDG_DATA_HOME")
		} else {
			os.Setenv("XDG_DATA_HOME", originalDataDir)
		}
	}()

	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)

	specsDir := filepath.Join(tmpDir, "collab-cli", "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	// First call should return 1
	num1, err := GetNextFeatureNumber()
	require.NoError(t, err)
	assert.Equal(t, 1, num1)

	// Second call should return 2
	num2, err := GetNextFeatureNumber()
	require.NoError(t, err)
	assert.Equal(t, 2, num2)

	// Third call should return 3
	num3, err := GetNextFeatureNumber()
	require.NoError(t, err)
	assert.Equal(t, 3, num3)
}

func TestGetNextFeatureNumber_AfterManualSpecCreation(t *testing.T) {
	// This tests the main bug fix: creating specs manually and ensuring
	// the counter syncs with existing directories
	originalDataDir := os.Getenv("XDG_DATA_HOME")
	defer func() {
		if originalDataDir == "" {
			os.Unsetenv("XDG_DATA_HOME")
		} else {
			os.Setenv("XDG_DATA_HOME", originalDataDir)
		}
	}()

	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)

	specsDir := filepath.Join(tmpDir, "collab-cli", "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	// Get first number
	num1, err := GetNextFeatureNumber()
	require.NoError(t, err)
	assert.Equal(t, 1, num1)

	// Simulate manual creation of spec directories (e.g., copied from another machine)
	require.NoError(t, os.MkdirAll(filepath.Join(specsDir, "005-manually-created"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(specsDir, "010-another-manual"), 0o755))

	// Next call should return 11 (max of counter=1, existing=10, then +1)
	num2, err := GetNextFeatureNumber()
	require.NoError(t, err)
	assert.Equal(t, 11, num2)
}
