package logging

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

func TestAssistantEventLogger(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := ioutil.TempDir("", "assistant-logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	// Create EventBus
	eventBus := events.NewEventBus(100)

	// Create AssistantEventLogger
	logger, err := NewAssistantEventLogger()
	if err != nil {
		t.Fatalf("Failed to create assistant logger: %v", err)
	}
	defer logger.Close()

	// Start logging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Start(ctx, eventBus)

	// Publish test assistant events
	sessionID := "test-session-123"

	// Request event
	requestEvent := events.NewAssistantRequest(
		sessionID,
		"test-orchestrator",
		"agent-123",
		"conv-456",
		"Test user message",
		map[string]interface{}{
			"test_flag": true,
		},
	)
	eventBus.Publish(requestEvent)

	// Response event
	responseEvent := events.NewAssistantResponse(
		sessionID,
		"test-orchestrator",
		"agent-123",
		"conv-456",
		"Test assistant response",
		150, // tokens
		1500, // timing ms
		"claude-3", // model
		map[string]interface{}{
			"test_response": true,
		},
	)
	eventBus.Publish(responseEvent)

	// Error event
	errorEvent := events.NewAssistantError(
		sessionID,
		"test-orchestrator",
		"agent-123",
		"test_error",
		"Test error message",
		"Test context",
		"retry",
		map[string]interface{}{
			"test_error": true,
		},
	)
	eventBus.Publish(errorEvent)

	// Metadata event
	metadataEvent := events.NewAssistantMetadata(
		sessionID,
		"test-orchestrator",
		"agent-123",
		200, // total tokens
		0.05, // cost
		2000, // duration ms
		"claude-3",
		map[string]interface{}{
			"performance": "good",
		},
		map[string]interface{}{
			"test_metadata": true,
		},
	)
	eventBus.Publish(metadataEvent)

	// Give some time for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Close logger to flush
	logger.Close()

	// Read the log file and verify content
	logPath := filepath.Join(tempDir, "collab-cli", "assistant.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify we have JSON lines for each event type
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 log lines, got %d", len(lines))
	}

	// Check that each line contains expected event types
	expectedTypes := []string{
		"assistant.request",
		"assistant.response",
		"assistant.error",
		"assistant.metadata",
	}

	for i, line := range lines {
		if i < len(expectedTypes) {
			expectedType := expectedTypes[i]
			if !strings.Contains(line, expectedType) {
				t.Errorf("Line %d should contain %s, but got: %s", i+1, expectedType, line)
			}

			// Verify it's valid JSON by checking basic structure
			if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
				t.Errorf("Line %d should be valid JSON: %s", i+1, line)
			}

			// Verify session ID is present
			if !strings.Contains(line, sessionID) {
				t.Errorf("Line %d should contain session ID %s: %s", i+1, sessionID, line)
			}
		}
	}

	t.Logf("Assistant log content:\n%s", logContent)
}

func TestAssistantEventLoggerFiltering(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := ioutil.TempDir("", "assistant-logger-filter-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	// Create EventBus
	eventBus := events.NewEventBus(100)

	// Create AssistantEventLogger
	logger, err := NewAssistantEventLogger()
	if err != nil {
		t.Fatalf("Failed to create assistant logger: %v", err)
	}
	defer logger.Close()

	// Start logging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Start(ctx, eventBus)

	// Publish non-assistant events (should be filtered out)
	sessionEvent := events.NewSessionCreated(&types.Session{
		ID: "test-session",
	})
	eventBus.Publish(sessionEvent)

	// Publish assistant event (should be logged)
	requestEvent := events.NewAssistantRequest(
		"test-session-123",
		"test-orchestrator",
		"agent-123",
		"conv-456",
		"Test message",
		nil,
	)
	eventBus.Publish(requestEvent)

	// Give some time for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Close logger to flush
	logger.Close()

	// Read the log file and verify only assistant events are logged
	logPath := filepath.Join(tempDir, "collab-cli", "assistant.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")

	// Should only have 1 line (the assistant request)
	if len(lines) != 1 {
		t.Errorf("Expected 1 log line (assistant events only), got %d", len(lines))
	}

	// Verify it contains the assistant request
	if !strings.Contains(logContent, "assistant.request") {
		t.Errorf("Log should contain assistant.request event")
	}

	// Verify it does not contain session events
	if strings.Contains(logContent, "session.created") {
		t.Errorf("Log should not contain non-assistant events")
	}

	t.Logf("Filtered assistant log content:\n%s", logContent)
}