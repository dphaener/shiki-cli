package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Message tests
func TestWriteAndReadMessages(t *testing.T) {
	wsDir := t.TempDir()

	// Create messages directory
	err := os.MkdirAll(filepath.Join(wsDir, "messages"), 0700)
	require.NoError(t, err)

	now := time.Now()

	msg1 := &types.Message{
		From:      "agent_1",
		To:        "agent_2",
		Turn:      0,
		Content:   "Hello from agent 1",
		Timestamp: now,
	}

	msg2 := &types.Message{
		From:      "agent_2",
		To:        "agent_1",
		Turn:      1,
		Content:   "Hello from agent 2",
		Timestamp: now.Add(1 * time.Minute),
	}

	// Write messages
	err = WriteMessage(wsDir, msg1)
	require.NoError(t, err)
	err = WriteMessage(wsDir, msg2)
	require.NoError(t, err)

	// Verify file paths set
	assert.NotEmpty(t, msg1.FilePath)
	assert.NotEmpty(t, msg1.ID)

	// Read messages for agent_2
	messages, err := ReadMessages(wsDir, "agent_2", 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "agent_1", messages[0].From)
	assert.Equal(t, "Hello from agent 1", messages[0].Content)

	// Read messages for agent_1
	messages, err = ReadMessages(wsDir, "agent_1", 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "agent_2", messages[0].From)
}

func TestReadMessagesSinceTurn(t *testing.T) {
	wsDir := t.TempDir()

	err := os.MkdirAll(filepath.Join(wsDir, "messages"), 0700)
	require.NoError(t, err)

	now := time.Now()

	// Write messages from turns 0, 1, 2
	for i := 0; i < 3; i++ {
		msg := &types.Message{
			From:      "agent_1",
			To:        "agent_2",
			Turn:      i,
			Content:   "Message",
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		}
		err := WriteMessage(wsDir, msg)
		require.NoError(t, err)
	}

	// Read only messages since turn 1
	messages, err := ReadMessages(wsDir, "agent_2", 1)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, 1, messages[0].Turn)
	assert.Equal(t, 2, messages[1].Turn)
}

func TestReadMessagesEmptyDir(t *testing.T) {
	wsDir := t.TempDir()

	messages, err := ReadMessages(wsDir, "agent_1", 0)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

// Context tests
func TestWriteAndReadSharedContext(t *testing.T) {
	wsDir := t.TempDir()

	content := "# Shared Context\n\nThis is shared information."
	err := WriteSharedContext(wsDir, content)
	require.NoError(t, err)

	readContent, err := ReadSharedContext(wsDir)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestReadSharedContextMissing(t *testing.T) {
	wsDir := t.TempDir()

	content, err := ReadSharedContext(wsDir)
	require.NoError(t, err)
	assert.Empty(t, content)
}

// Memory tests
func TestWriteAndReadMemory(t *testing.T) {
	wsDir := t.TempDir()

	err := os.MkdirAll(filepath.Join(wsDir, "memory"), 0700)
	require.NoError(t, err)

	agentID := "agent_1"
	content := "# Agent 1 Memory\n\nRemembered facts..."

	err = WriteMemory(wsDir, agentID, content)
	require.NoError(t, err)

	readContent, err := ReadMemory(wsDir, agentID)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestReadMemoryMissing(t *testing.T) {
	wsDir := t.TempDir()

	content, err := ReadMemory(wsDir, "agent_1")
	require.NoError(t, err)
	assert.Empty(t, content)
}

func TestMemoryIsolation(t *testing.T) {
	wsDir := t.TempDir()

	err := os.MkdirAll(filepath.Join(wsDir, "memory"), 0700)
	require.NoError(t, err)

	// Write different content for each agent
	err = WriteMemory(wsDir, "agent_1", "Agent 1 memory")
	require.NoError(t, err)
	err = WriteMemory(wsDir, "agent_2", "Agent 2 memory")
	require.NoError(t, err)

	// Verify isolation
	content1, err := ReadMemory(wsDir, "agent_1")
	require.NoError(t, err)
	content2, err := ReadMemory(wsDir, "agent_2")
	require.NoError(t, err)

	assert.Equal(t, "Agent 1 memory", content1)
	assert.Equal(t, "Agent 2 memory", content2)
}

// Deliverable tests
func TestWriteAndReadDeliverable(t *testing.T) {
	wsDir := t.TempDir()
	now := time.Now()

	content := "# Final Deliverable\n\nThis is the result."
	metadata := DeliverableMetadata{
		SubmittedBy: "agent_1",
		SubmittedAt: now,
		ApprovedBy:  []string{"agent_2"},
	}

	err := WriteDeliverable(wsDir, content, metadata)
	require.NoError(t, err)

	readContent, readMeta, err := ReadDeliverable(wsDir)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
	assert.Equal(t, metadata.SubmittedBy, readMeta.SubmittedBy)
	assert.Len(t, readMeta.ApprovedBy, 1)
}

func TestReadDeliverableNoMetadata(t *testing.T) {
	wsDir := t.TempDir()

	// Write file without metadata
	path := filepath.Join(wsDir, "deliverable.md")
	err := os.WriteFile(path, []byte("Plain content"), 0600)
	require.NoError(t, err)

	content, meta, err := ReadDeliverable(wsDir)
	require.NoError(t, err)
	assert.Equal(t, "Plain content", content)
	assert.Nil(t, meta)
}

// Integration test
func TestStorageIntegration(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "20251123-120000-test"

	// Create workspace
	structure := []WorkspaceStructure{
		{Path: "docs", Type: "directory"},
	}
	wsDir, err := CreateWorkspace(sessionID, baseDir, structure)
	require.NoError(t, err)

	// Create session
	session := &types.Session{
		ID:           sessionID,
		WorkspaceDir: wsDir,
		Status:       types.SessionRunning,
		CreatedAt:    time.Now(),
		CurrentTurn:  0,
		MaxTurns:     10,
	}

	// Save state
	err = SaveSessionState(session)
	require.NoError(t, err)

	// Write message
	msg := &types.Message{
		From:      "agent_1",
		To:        "agent_2",
		Turn:      0,
		Content:   "Hello",
		Timestamp: time.Now(),
	}
	err = WriteMessage(wsDir, msg)
	require.NoError(t, err)

	// Read messages
	messages, err := ReadMessages(wsDir, "agent_2", 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)

	// Write context
	err = WriteSharedContext(wsDir, "# Context\n")
	require.NoError(t, err)

	// Write memory
	err = WriteMemory(wsDir, "agent_1", "# Memory\n")
	require.NoError(t, err)

	// Cleanup
	err = CleanupWorkspace(wsDir)
	require.NoError(t, err)
	assert.NoDirExists(t, wsDir)
}
