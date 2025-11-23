package logging

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBusLoggerIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	// Start event logger
	eventLogger, err := NewEventLogger(tmpDir)
	require.NoError(t, err)
	defer eventLogger.Close()

	ctx, cancel := context.WithCancel(context.Background())
	eventLogger.Start(ctx, bus)
	defer cancel()

	// Create test session
	session := &types.Session{
		ID:     "test-session",
		Status: types.SessionRunning,
	}

	turn := &types.Turn{
		Number:     1,
		AgentID:    "agent-1",
		DurationMS: 100,
	}

	// Publish various events
	bus.Publish(events.NewSessionCreated(session))
	bus.Publish(events.NewSessionStarted(session))
	bus.Publish(events.NewTurnStarted(turn, "agent-1", "test-session"))
	bus.Publish(events.NewTurnCompleted(turn, "test-session"))

	// Give logger time to write
	time.Sleep(100 * time.Millisecond)

	// Verify log file exists and has correct permissions
	logPath := filepath.Join(tmpDir, "orchestrator.log")
	info, err := os.Stat(logPath)
	require.NoError(t, err, "log file should exist")
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "log file should have 0600 permissions")

	// Read and parse log file
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.GreaterOrEqual(t, len(lines), 4, "should have at least 4 log entries")

	// Parse first log entry
	var entry map[string]interface{}
	err = json.Unmarshal([]byte(lines[0]), &entry)
	require.NoError(t, err, "log entry should be valid JSON")

	assert.Equal(t, "info", entry["level"])
	assert.Contains(t, entry["message"], "SessionCreated")

	// Verify all entries have required fields
	for i, line := range lines {
		if line == "" {
			continue
		}
		var e map[string]interface{}
		err := json.Unmarshal([]byte(line), &e)
		require.NoError(t, err, "line %d should be valid JSON", i)

		assert.NotEmpty(t, e["timestamp"], "line %d should have timestamp", i)
		assert.NotEmpty(t, e["level"], "line %d should have level", i)
		assert.NotEmpty(t, e["message"], "line %d should have message", i)

		// Check fields structure
		if fields, ok := e["fields"].(map[string]interface{}); ok {
			assert.NotEmpty(t, fields["event_type"], "line %d should have event_type", i)
			assert.NotEmpty(t, fields["session_id"], "line %d should have session_id", i)
		}
	}
}

func TestEventLoggerFileAppend(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial log entry
	logPath := filepath.Join(tmpDir, "orchestrator.log")
	err := os.WriteFile(logPath, []byte("existing content\n"), 0600)
	require.NoError(t, err)

	// Create event logger (should append, not overwrite)
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	eventLogger, err := NewEventLogger(tmpDir)
	require.NoError(t, err)
	defer eventLogger.Close()

	ctx, cancel := context.WithCancel(context.Background())
	eventLogger.Start(ctx, bus)
	defer cancel()

	// Publish event
	session := &types.Session{ID: "test", Status: types.SessionRunning}
	bus.Publish(events.NewSessionCreated(session))

	time.Sleep(50 * time.Millisecond)

	// Verify file was appended to, not overwritten
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "existing content", "should preserve existing content")
	assert.Contains(t, content, "SessionCreated", "should append new content")
}

func TestEventLoggerGracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()

	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	eventLogger, err := NewEventLogger(tmpDir)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	eventLogger.Start(ctx, bus)

	// Publish some events
	session := &types.Session{ID: "test", Status: types.SessionRunning}
	for i := 0; i < 10; i++ {
		bus.Publish(events.NewSessionCreated(session))
	}

	// Cancel context (graceful shutdown)
	cancel()

	// Give goroutine time to shutdown
	time.Sleep(100 * time.Millisecond)

	// Should be able to close without error (idempotent)
	err = eventLogger.Close()
	assert.NoError(t, err)
}
