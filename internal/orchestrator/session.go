package orchestrator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/template"
	"github.com/dphaener/shiki-cli/internal/templates"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// NewSession creates a new collaboration session from a template
func NewSession(tmpl *template.TaskTemplate, templatePath, workspaceDir string) (*types.Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}

	now := time.Now()
	taskName := extractTaskName(tmpl)

	session := &types.Session{
		ID:               sessionID,
		TaskName:         taskName,
		TaskTemplatePath: templatePath,
		WorkspaceDir:     workspaceDir,
		Status:           types.SessionRunning,
		CreatedAt:        now,
		StartedAt:        &now,
		CurrentTurn:      0,
		MaxTurns:         tmpl.MaxTurns,
		Agent1:           agentFromTemplate(tmpl, "agent_1", workspaceDir),
		Agent2:           agentFromTemplate(tmpl, "agent_2", workspaceDir),
		TurnHistory:      []types.Turn{},
		TotalCost:        0.0,
		TotalTokens:      0,
	}

	return session, nil
}

// LoadSession loads a session from its workspace directory
func LoadSession(workspaceDir string) (*types.Session, error) {
	session, err := storage.LoadSessionState(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("load session state: %w", err)
	}

	return session, nil
}

// SaveSession persists the session to disk
func SaveSession(session *types.Session) error {
	if err := storage.SaveSessionState(session); err != nil {
		return fmt.Errorf("save session state: %w", err)
	}

	return nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// extractTaskName extracts a human-readable task name from the template
func extractTaskName(tmpl *template.TaskTemplate) string {
	// Try to extract from task body if available
	if tmpl.TaskBody != "" {
		// Take first line or first 50 chars
		lines := strings.Split(tmpl.TaskBody, "\n")
		name := lines[0]
		if len(name) > 50 {
			name = name[:50] + "..."
		}
		return name
	}

	// Fallback to a generic name
	return "Collaboration Task"
}

// agentFromTemplate creates an Agent configuration from template
func agentFromTemplate(tmpl *template.TaskTemplate, agentID, workspaceDir string) types.Agent {
	var name, role, model string

	if agentID == "agent_1" {
		name = tmpl.Agent1Name
		role = tmpl.Agent1Role
		model = tmpl.Agent1Model
	} else {
		name = tmpl.Agent2Name
		role = tmpl.Agent2Role
		model = tmpl.Agent2Model
	}

	// Default model if not specified
	if model == "" {
		model = "claude-sonnet-4"
	}

	// Use TemplateProcessor to load base collaboration system prompt
	// Note: The actual dynamic prompt is built in buildQuery() for each turn
	processor := templates.NewTemplateProcessor("templates")
	context := templates.NewPhaseContext().
		WithCore("Collaboration Base", 0, "", "collaboration").
		WithCollaborationData(name, role, agentID, "", 1, 10, "", "", "", "")

	baseSystemPrompt, err := processor.LoadSystemPrompt("collaboration", context)
	if err != nil {
		// Fallback if template loading fails
		baseSystemPrompt = fmt.Sprintf("You are %s, a %s. You will collaborate with another agent to complete tasks.", name, role)
	}

	return types.Agent{
		ID:           agentID,
		Name:         name,
		Role:         role,
		SystemPrompt: baseSystemPrompt,
		Model:        model,
		WorkspaceDir: workspaceDir,
		MemoryFile:   fmt.Sprintf("%s_memory.md", agentID),
		TotalTurns:   0,
		TotalTokens:  0,
		TotalCost:    0.0,
	}
}

// ResumeSession validates and prepares a session for resumption
func ResumeSession(session *types.Session) error {
	if session.Status != types.SessionPaused {
		return fmt.Errorf("session %s is not paused (status: %s)", session.ID, session.Status)
	}

	// Update status to running
	session.Status = types.SessionRunning

	return nil
}

// PauseSession gracefully pauses a session and saves state
func PauseSession(session *types.Session) error {
	session.Status = types.SessionPaused

	if err := SaveSession(session); err != nil {
		return fmt.Errorf("save paused session: %w", err)
	}

	return nil
}

// CompleteSession marks a session as completed and saves final state
func CompleteSession(session *types.Session, deliverablePath string) error {
	now := time.Now()
	session.Status = types.SessionCompleted
	session.CompletedAt = &now
	session.DeliverablePath = deliverablePath

	if err := SaveSession(session); err != nil {
		return fmt.Errorf("save completed session: %w", err)
	}

	return nil
}

// MarkIncomplete marks a session as incomplete (max turns reached without deliverable)
func MarkIncomplete(session *types.Session) error {
	now := time.Now()
	session.Status = types.SessionIncomplete
	session.CompletedAt = &now

	if err := SaveSession(session); err != nil {
		return fmt.Errorf("save incomplete session: %w", err)
	}

	return nil
}

// MarkError marks a session as errored and saves state
func MarkError(session *types.Session, errorMsg string) error {
	now := time.Now()
	session.Status = types.SessionError
	session.CompletedAt = &now

	if err := SaveSession(session); err != nil {
		return fmt.Errorf("save error session: %w", err)
	}

	return nil
}
