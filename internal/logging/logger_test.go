package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelWarn, &buf)

	// Debug and Info should be filtered out
	logger.Debug("debug message", nil)
	logger.Info("info message", nil)
	assert.Empty(t, buf.String(), "debug and info should be filtered")

	// Warn should be logged
	logger.Warn("warn message", nil)
	assert.NotEmpty(t, buf.String(), "warn should be logged")

	// Error should be logged
	buf.Reset()
	logger.Error("error message", nil)
	assert.NotEmpty(t, buf.String(), "error should be logged")
}

func TestLoggerJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	logger.Info("test message", fields)

	// Parse JSON
	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "info", entry.Level)
	assert.Equal(t, "test message", entry.Message)
	assert.NotEmpty(t, entry.Timestamp)
	assert.Equal(t, "value1", entry.Fields["key1"])
	assert.Equal(t, float64(42), entry.Fields["key2"])
	assert.Equal(t, true, entry.Fields["key3"])
}

func TestLoggerAllLevels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*Logger)
		expected string
	}{
		{
			name: "debug",
			logFunc: func(l *Logger) {
				l.Debug("debug msg", nil)
			},
			expected: "debug",
		},
		{
			name: "info",
			logFunc: func(l *Logger) {
				l.Info("info msg", nil)
			},
			expected: "info",
		},
		{
			name: "warn",
			logFunc: func(l *Logger) {
				l.Warn("warn msg", nil)
			},
			expected: "warn",
		},
		{
			name: "error",
			logFunc: func(l *Logger) {
				l.Error("error msg", nil)
			},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(LevelDebug, &buf)

			tt.logFunc(logger)

			var entry logEntry
			err := json.Unmarshal(buf.Bytes(), &entry)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, entry.Level)
		})
	}
}

func TestLoggerNilFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	logger.Info("test message", nil)

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Nil(t, entry.Fields)
}

func TestLoggerNewlineAppended(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	logger.Info("message 1", nil)
	logger.Info("message 2", nil)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2, "should have 2 lines")

	// Each line should be valid JSON
	for i, line := range lines {
		var entry logEntry
		err := json.Unmarshal([]byte(line), &entry)
		require.NoError(t, err, "line %d should be valid JSON", i)
	}
}

func TestLoggerDefaultOutput(t *testing.T) {
	// Logger with nil output should default to os.Stdout
	logger := NewLogger(LevelInfo, nil)
	assert.NotNil(t, logger.output)
}
