package mcp

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/storage"
	"github.com/darinhaener/collab/pkg/types"
)

// HandleSendMessage implements the send_message tool (T027)
func (s *Server) HandleSendMessage(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required and must be a non-empty string")
	}

	// Determine recipient
	recipientID := "agent_2"
	if agentID == "agent_2" {
		recipientID = "agent_1"
	}

	// Create message
	msg := &types.Message{
		From:      agentID,
		To:        recipientID,
		Turn:      turn,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Write via storage layer (atomic)
	if err := storage.WriteMessage(s.workspaceDir, msg); err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("send_message", agentID, s.sessionID, turn, args))

	// Emit MessageSent event
	s.eventBus.Publish(events.NewMessageSent(msg, s.sessionID))

	return map[string]interface{}{
		"message_path": msg.FilePath,
		"message_id":   msg.ID,
	}, nil
}

// HandleReadMessages implements the read_messages tool (T027)
func (s *Server) HandleReadMessages(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	sinceTurn := 0
	if st, ok := args["since_turn"].(float64); ok {
		sinceTurn = int(st)
	} else if st, ok := args["since_turn"].(int); ok {
		sinceTurn = st
	}

	// Read messages from storage
	messages, err := storage.ReadMessages(s.workspaceDir, agentID, sinceTurn)
	if err != nil {
		return nil, fmt.Errorf("read messages: %w", err)
	}

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("read_messages", agentID, s.sessionID, turn, args))

	// Convert to response format
	messageList := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageList[i] = map[string]interface{}{
			"id":        msg.ID,
			"from":      msg.From,
			"to":        msg.To,
			"timestamp": msg.Timestamp.Format(time.RFC3339),
			"turn":      msg.Turn,
			"content":   msg.Content,
			"file_path": msg.FilePath,
		}
	}

	return map[string]interface{}{
		"messages": messageList,
	}, nil
}

// HandleWriteSharedContext implements the write_shared_context tool (T028)
func (s *Server) HandleWriteSharedContext(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required")
	}

	if err := storage.WriteSharedContext(s.workspaceDir, content); err != nil {
		return nil, fmt.Errorf("write shared context: %w", err)
	}

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("write_shared_context", agentID, s.sessionID, turn, args))

	// Emit FileUpdated event
	path := filepath.Join(s.workspaceDir, "shared_context.md")
	s.eventBus.Publish(events.NewFileUpdated(path, "modified", s.sessionID))

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// HandleReadSharedContext implements the read_shared_context tool (T028)
func (s *Server) HandleReadSharedContext(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	content, err := storage.ReadSharedContext(s.workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("read shared context: %w", err)
	}

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("read_shared_context", agentID, s.sessionID, turn, args))

	return map[string]interface{}{
		"content": content,
		"path":    filepath.Join(s.workspaceDir, "shared_context.md"),
	}, nil
}

// HandleUpdateMemory implements the update_memory tool (T029)
func (s *Server) HandleUpdateMemory(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required")
	}

	if err := storage.WriteMemory(s.workspaceDir, agentID, content); err != nil {
		return nil, fmt.Errorf("write memory: %w", err)
	}

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("update_memory", agentID, s.sessionID, turn, args))

	path := filepath.Join(s.workspaceDir, "memory", fmt.Sprintf("%s_memory.md", agentID))
	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// HandleReadMemory implements the read_memory tool (T029)
func (s *Server) HandleReadMemory(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	content, err := storage.ReadMemory(s.workspaceDir, agentID)
	if err != nil {
		return nil, fmt.Errorf("read memory: %w", err)
	}

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("read_memory", agentID, s.sessionID, turn, args))

	return map[string]interface{}{
		"content": content,
		"path":    filepath.Join(s.workspaceDir, "memory", fmt.Sprintf("%s_memory.md", agentID)),
	}, nil
}

// HandleSubmitDeliverable implements the submit_deliverable tool (T030)
func (s *Server) HandleSubmitDeliverable(agentID string, turn int, args map[string]interface{}) (map[string]interface{}, error) {
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required and must be non-empty")
	}

	approved, ok := args["approved"].(bool)
	if !ok || !approved {
		return nil, fmt.Errorf("approved must be true")
	}

	// Store this agent's deliverable (thread-safe)
	s.deliverablesMu.Lock()
	s.deliverables[agentID] = content
	otherAgentID := "agent_2"
	if agentID == "agent_2" {
		otherAgentID = "agent_1"
	}
	otherContent, hasOther := s.deliverables[otherAgentID]
	s.deliverablesMu.Unlock()

	// Emit ToolInvoked event
	s.eventBus.Publish(events.NewToolInvoked("submit_deliverable", agentID, s.sessionID, turn, args))

	if hasOther {
		// Both submitted - check if content matches byte-for-byte
		if content == otherContent {
			// Content matches - write deliverable and emit completion event
			metadata := storage.DeliverableMetadata{
				SubmittedBy: agentID,
				SubmittedAt: time.Now(),
				ApprovedBy:  []string{"agent_1", "agent_2"},
			}

			if err := storage.WriteDeliverable(s.workspaceDir, content, metadata); err != nil {
				return nil, fmt.Errorf("write deliverable: %w", err)
			}

			// Emit SessionCompleted event without Session object (we don't have it in MCP server)
			s.eventBus.Publish(events.Event{
				Type:      types.EventSessionCompleted,
				SessionID: s.sessionID,
				Timestamp: time.Now(),
				Payload: events.SessionCompletedPayload{
					Session:         nil, // Will be populated by orchestrator
					DeliverablePath: filepath.Join(s.workspaceDir, "deliverable.md"),
				},
			})

			return map[string]interface{}{
				"status":  "completed",
				"message": "Both agents approved matching deliverables. Session complete!",
			}, nil
		}
		// Content mismatch
		return map[string]interface{}{
			"status":  "pending",
			"message": "Deliverable content does not match other agent's submission. Continue collaborating.",
		}, nil
	}

	// Only this agent has submitted so far
	return map[string]interface{}{
		"status":  "pending",
		"message": fmt.Sprintf("Deliverable submitted. Waiting for %s to submit.", otherAgentID),
	}, nil
}
