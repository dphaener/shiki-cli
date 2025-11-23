package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/template"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// WrapCommandForTest wraps a command with a root command that has global flags
func WrapCommandForTest(cmd *cobra.Command) *cobra.Command {
	root := &cobra.Command{
		Use: "collab",
	}

	// Add global flags like the real root command
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	root.PersistentFlags().Bool("no-color", false, "Disable colored output")
	root.PersistentFlags().StringP("config", "c", "", "Config file path")
	root.PersistentFlags().StringP("workspace", "w", "", "Workspace directory")

	root.AddCommand(cmd)

	return root
}

// CaptureOutput captures stdout and stderr from a function
func CaptureOutput(fn func()) string {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	fn()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// ExecuteCommand executes a command and returns output and error
func ExecuteCommand(root *cobra.Command, args ...string) (string, error) {
	var output string

	root.SetArgs(args)

	output = CaptureOutput(func() {
		root.Execute()
	})

	// Note: We capture the output but return the error from the actual execution
	root.SetArgs(args)
	err := root.Execute()

	return output, err
}

// createTempTemplate creates a temporary template file for testing
func createTempTemplate(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "template.md")

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	require.NoError(t, err)

	return tmpFile
}

// createValidTemplate creates a valid template for testing
func createValidTemplate(t *testing.T) string {
	t.Helper()

	content := `---
agent1_name: "Alice"
agent1_role: "Proposer"
agent1_system_prompt: "You are Alice, a proposal writer."
agent2_name: "Bob"
agent2_role: "Reviewer"
agent2_system_prompt: "You are Bob, a careful reviewer."
max_turns: 10
---
# Collaboration Task

Write a project proposal and review it.

## Agent 1 Instructions

Draft a project proposal.

## Agent 2 Instructions

Review the proposal and provide feedback.
`

	return createTempTemplate(t, content)
}

// createTestSession creates a test session in a temporary directory
func createTestSession(t *testing.T, workspaceRoot, sessionID string, status types.SessionStatus) *types.Session {
	t.Helper()

	// Create session directory
	sessionDir := filepath.Join(workspaceRoot, sessionID)
	err := os.MkdirAll(sessionDir, 0700)
	require.NoError(t, err)

	// Create a simple template
	tmpl := &template.TaskTemplate{
		Agent1Name:         "Agent1",
		Agent1Role:         "Role1",
		Agent1SystemPrompt: "Prompt1",
		Agent2Name:         "Agent2",
		Agent2Role:         "Role2",
		Agent2SystemPrompt: "Prompt2",
		MaxTurns:           10,
	}

	// Create session
	session, err := orchestrator.NewSession(tmpl, "test-template.md", sessionDir)
	require.NoError(t, err)

	// Override session ID
	session.ID = sessionID
	session.Status = status
	session.WorkspaceDir = sessionDir

	// Set timestamps
	now := time.Now()
	session.StartedAt = &now

	if status == types.SessionCompleted {
		completedAt := now.Add(5 * time.Minute)
		session.CompletedAt = &completedAt
	}

	// Set some metrics
	session.CurrentTurn = 5
	session.TotalCost = 0.25

	// Save session state
	err = orchestrator.SaveSession(session)
	require.NoError(t, err)

	return session
}
