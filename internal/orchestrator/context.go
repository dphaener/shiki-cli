package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/darinhaener/collab/internal/storage"
)

// ReadTask reads the task.md file from the workspace
func (o *Orchestrator) ReadTask() (string, error) {
	path := filepath.Join(o.session.WorkspaceDir, "task.md")
	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read task.md: %w", err)
	}
	return string(data), nil
}

// ReadPartnerMessages reads all messages addressed to the given agent
// Returns formatted string with all messages for injection into query
func (o *Orchestrator) ReadPartnerMessages(agentID string) (string, error) {
	messages, err := storage.ReadMessages(o.session.WorkspaceDir, agentID, 0)
	if err != nil {
		return "", fmt.Errorf("read messages: %w", err)
	}

	if len(messages) == 0 {
		return "(No messages yet)", nil
	}

	var sb strings.Builder
	for i, msg := range messages {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		sb.WriteString(fmt.Sprintf("**From %s (Turn %d):**\n%s\n", msg.From, msg.Turn, msg.Content))
	}

	return sb.String(), nil
}

// ReadSharedContext reads the shared context file
func (o *Orchestrator) ReadSharedContext() (string, error) {
	content, err := storage.ReadSharedContext(o.session.WorkspaceDir)
	if err != nil {
		return "", fmt.Errorf("read shared context: %w", err)
	}

	if content == "" || content == "# Shared Context\n\n" {
		return "(No shared context yet)", nil
	}

	return content, nil
}

// ReadAgentMemory reads an agent's private memory file
func (o *Orchestrator) ReadAgentMemory(agentID string) (string, error) {
	content, err := storage.ReadMemory(o.session.WorkspaceDir, agentID)
	if err != nil {
		return "", fmt.Errorf("read agent memory: %w", err)
	}

	if content == "" || content == fmt.Sprintf("# %s Memory\n\n", agentID) {
		return "(No memory saved yet)", nil
	}

	return content, nil
}

// GetPartnerID returns the partner agent's ID
func (o *Orchestrator) GetPartnerID(agentID string) string {
	if agentID == o.session.Agent1.ID {
		return o.session.Agent2.ID
	}
	return o.session.Agent1.ID
}

// GetPartnerName returns the partner agent's name
func (o *Orchestrator) GetPartnerName(agentID string) string {
	if agentID == o.session.Agent1.ID {
		return o.session.Agent2.Name
	}
	return o.session.Agent1.Name
}
