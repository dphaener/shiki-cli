package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/darinhaener/collab/internal/events"
)

func TestMCPServerIntegration(t *testing.T) {
	wsDir := t.TempDir()

	// Create required directories
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, "messages"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, "memory"), 0o700))

	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	server := NewServer(wsDir, "test-session", bus)
	err := server.Start()
	require.NoError(t, err)
	defer server.Shutdown()

	// Test 1: send_message
	t.Run("send_message", func(t *testing.T) {
		result, err := server.HandleSendMessage("agent_1", 0, map[string]interface{}{
			"content": "Hello from agent 1",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "message_path")
		assert.Contains(t, result, "message_id")

		// Verify file was created
		messagePath := result["message_path"].(string)
		assert.FileExists(t, messagePath)
	})

	// Test 2: read_messages
	t.Run("read_messages", func(t *testing.T) {
		result, err := server.HandleReadMessages("agent_2", 0, map[string]interface{}{})
		require.NoError(t, err)

		messages := result["messages"].([]map[string]interface{})
		assert.Len(t, messages, 1)
		assert.Equal(t, "agent_1", messages[0]["from"])
		assert.Equal(t, "agent_2", messages[0]["to"])
		assert.Equal(t, "Hello from agent 1", messages[0]["content"])
	})

	// Test 3: read_messages with since_turn filter
	t.Run("read_messages_with_filter", func(t *testing.T) {
		// Send another message at turn 1
		_, err := server.HandleSendMessage("agent_1", 1, map[string]interface{}{
			"content": "Second message",
		})
		require.NoError(t, err)

		// Read only from turn 1
		result, err := server.HandleReadMessages("agent_2", 1, map[string]interface{}{
			"since_turn": 1,
		})
		require.NoError(t, err)

		messages := result["messages"].([]map[string]interface{})
		assert.Len(t, messages, 1)
		assert.Equal(t, "Second message", messages[0]["content"])
	})

	// Test 4: write_shared_context
	t.Run("write_shared_context", func(t *testing.T) {
		result, err := server.HandleWriteSharedContext("agent_1", 0, map[string]interface{}{
			"content": "# Shared Context\n\nThis is our shared workspace.",
		})
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result["path"], "shared_context.md")

		// Verify file was created
		assert.FileExists(t, result["path"].(string))
	})

	// Test 5: read_shared_context
	t.Run("read_shared_context", func(t *testing.T) {
		result, err := server.HandleReadSharedContext("agent_2", 0, map[string]interface{}{})
		require.NoError(t, err)

		content := result["content"].(string)
		assert.Equal(t, "# Shared Context\n\nThis is our shared workspace.", content)
	})

	// Test 6: update_memory
	t.Run("update_memory", func(t *testing.T) {
		result, err := server.HandleUpdateMemory("agent_1", 0, map[string]interface{}{
			"content": "# Agent 1 Memory\n\nI remember this.",
		})
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result["path"], "agent_1_memory.md")

		// Verify file was created
		assert.FileExists(t, result["path"].(string))
	})

	// Test 7: read_memory
	t.Run("read_memory", func(t *testing.T) {
		result, err := server.HandleReadMemory("agent_1", 0, map[string]interface{}{})
		require.NoError(t, err)

		content := result["content"].(string)
		assert.Equal(t, "# Agent 1 Memory\n\nI remember this.", content)
	})

	// Test 8: memory isolation (agent_2 can't read agent_1's memory)
	t.Run("memory_isolation", func(t *testing.T) {
		result, err := server.HandleReadMemory("agent_2", 0, map[string]interface{}{})
		require.NoError(t, err)

		content := result["content"].(string)
		assert.Empty(t, content, "agent_2 should not see agent_1's memory")
	})

	// Test 9: submit_deliverable - first agent
	t.Run("submit_deliverable_first", func(t *testing.T) {
		deliverableContent := "# Final Design\n\nThis is our agreed solution."

		result, err := server.HandleSubmitDeliverable("agent_1", 1, map[string]interface{}{
			"content":  deliverableContent,
			"approved": true,
		})
		require.NoError(t, err)
		assert.Equal(t, "pending", result["status"])
		assert.Contains(t, result["message"], "Waiting for agent_2")
	})

	// Test 10: submit_deliverable - both agents with matching content
	t.Run("submit_deliverable_completion", func(t *testing.T) {
		deliverableContent := "# Final Design\n\nThis is our agreed solution."

		result, err := server.HandleSubmitDeliverable("agent_2", 1, map[string]interface{}{
			"content":  deliverableContent,
			"approved": true,
		})
		require.NoError(t, err)
		assert.Equal(t, "completed", result["status"])
		assert.Contains(t, result["message"], "Session complete")

		// Verify deliverable file was created
		deliverablePath := filepath.Join(wsDir, "deliverable.md")
		assert.FileExists(t, deliverablePath)
	})
}

func TestMCPServerDeliverableMismatch(t *testing.T) {
	wsDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, "messages"), 0o700))

	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	server := NewServer(wsDir, "test-session-2", bus)
	err := server.Start()
	require.NoError(t, err)
	defer server.Shutdown()

	// Agent 1 submits
	result, err := server.HandleSubmitDeliverable("agent_1", 1, map[string]interface{}{
		"content":  "# Design A\n\nMy version.",
		"approved": true,
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", result["status"])

	// Agent 2 submits different content
	result, err = server.HandleSubmitDeliverable("agent_2", 1, map[string]interface{}{
		"content":  "# Design B\n\nDifferent version.",
		"approved": true,
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", result["status"])
	assert.Contains(t, result["message"], "does not match")

	// Deliverable file should NOT be created
	deliverablePath := filepath.Join(wsDir, "deliverable.md")
	assert.NoFileExists(t, deliverablePath)
}

func TestMCPServerValidation(t *testing.T) {
	wsDir := t.TempDir()
	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	server := NewServer(wsDir, "test-session-3", bus)
	require.NoError(t, server.Start())
	defer server.Shutdown()

	// Test empty content validation
	t.Run("send_message_empty_content", func(t *testing.T) {
		_, err := server.HandleSendMessage("agent_1", 0, map[string]interface{}{
			"content": "",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty")
	})

	t.Run("send_message_missing_content", func(t *testing.T) {
		_, err := server.HandleSendMessage("agent_1", 0, map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("submit_deliverable_not_approved", func(t *testing.T) {
		_, err := server.HandleSubmitDeliverable("agent_1", 1, map[string]interface{}{
			"content":  "Some content",
			"approved": false,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be true")
	})

	t.Run("submit_deliverable_empty_content", func(t *testing.T) {
		_, err := server.HandleSubmitDeliverable("agent_1", 1, map[string]interface{}{
			"content":  "",
			"approved": true,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty")
	})
}

func TestMCPServerShutdown(t *testing.T) {
	wsDir := t.TempDir()
	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	server := NewServer(wsDir, "test-session-4", bus)
	require.NoError(t, server.Start())

	// Verify context is active
	assert.NotNil(t, server.Context())

	// Shutdown
	err := server.Shutdown()
	require.NoError(t, err)

	// Context should be cancelled
	select {
	case <-server.Context().Done():
		// Expected
	default:
		t.Error("Context should be cancelled after shutdown")
	}
}

func TestGetTools(t *testing.T) {
	wsDir := t.TempDir()
	bus := events.NewEventBus(100)
	defer bus.Shutdown()

	server := NewServer(wsDir, "test-session-5", bus)

	tools := server.GetTools()
	assert.Len(t, tools, 7, "Should have exactly 7 tools")

	toolNames := []string{
		"send_message",
		"read_messages",
		"write_shared_context",
		"read_shared_context",
		"update_memory",
		"read_memory",
		"submit_deliverable",
	}

	for i, tool := range tools {
		assert.Equal(t, toolNames[i], tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
	}
}
