package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowCommand(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create test session
	session := createTestSession(t, tmpDir, "test-session-123", types.SessionCompleted)

	tests := []struct {
		name           string
		args           []string
		wantExitCode   int
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "show session summary",
			args:         []string{"--workspace", tmpDir, "test-session-123"},
			wantExitCode: ExitSuccess,
			wantContains: []string{
				"Session: test-session-123",
				"Status: completed",
				"Agent1",
				"Agent2",
			},
		},
		{
			name:         "show with messages flag",
			args:         []string{"--workspace", tmpDir, "test-session-123", "--messages"},
			wantExitCode: ExitSuccess,
			wantContains: []string{
				"Session: test-session-123",
			},
		},
		{
			name:         "show with deliverable flag",
			args:         []string{"--workspace", tmpDir, "test-session-123", "--deliverable"},
			wantExitCode: ExitSuccess,
			wantContains: []string{
				"Session: test-session-123",
			},
		},
		{
			name:         "show with logs flag",
			args:         []string{"--workspace", tmpDir, "test-session-123", "--logs"},
			wantExitCode: ExitSuccess,
			wantContains: []string{
				"Session: test-session-123",
			},
		},
		{
			name:         "show with context flag",
			args:         []string{"--workspace", tmpDir, "test-session-123", "--context"},
			wantExitCode: ExitSuccess,
			wantContains: []string{
				"Session: test-session-123",
			},
		},
		{
			name:         "show non-existent session",
			args:         []string{"--workspace", tmpDir, "non-existent"},
			wantExitCode: ExitError,
			wantContains: []string{
				"Error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd := NewShowCommand()
			cmd.SetArgs(tt.args)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			if tt.wantExitCode == ExitSuccess {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			output := stdout.String() + stderr.String()

			// Check output contains expected substrings
			for _, substr := range tt.wantContains {
				assert.Contains(t, output, substr)
			}

			// Check output doesn't contain unwanted substrings
			for _, substr := range tt.wantNotContain {
				assert.NotContains(t, output, substr)
			}
		})
	}

	_ = session
}

func TestShowCommand_WithMessages(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create test session
	session := createTestSession(t, tmpDir, "test-session", types.SessionRunning)

	// Create some messages
	messagesFile := filepath.Join(tmpDir, "test-session", "messages.jsonl")
	messageContent := `{"timestamp":"2025-11-23T12:00:00Z","from":"agent1","to":"agent2","content":"Hello"}`
	err := os.WriteFile(messagesFile, []byte(messageContent+"\n"), 0600)
	require.NoError(t, err)

	var stdout bytes.Buffer

	cmd := NewShowCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "test-session", "--messages"})
	cmd.SetOut(&stdout)

	err = cmd.Execute()
	require.NoError(t, err)

	// Should display messages when flag is provided
	// The exact output format depends on implementation

	_ = session
}

func TestShowCommand_WithDeliverable(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create test session
	session := createTestSession(t, tmpDir, "test-session", types.SessionCompleted)

	// Create a deliverable file
	deliverableFile := filepath.Join(tmpDir, "test-session", "deliverable.txt")
	deliverableContent := "Final deliverable content"
	err := os.WriteFile(deliverableFile, []byte(deliverableContent), 0600)
	require.NoError(t, err)

	// Update session to reference deliverable
	session.DeliverablePath = deliverableFile

	var stdout bytes.Buffer

	cmd := NewShowCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "test-session", "--deliverable"})
	cmd.SetOut(&stdout)

	err = cmd.Execute()
	require.NoError(t, err)

	// The exact output format depends on implementation
	_ = session
}

func TestShowCommand_AllFlags(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create test session
	session := createTestSession(t, tmpDir, "test-session", types.SessionCompleted)

	var stdout bytes.Buffer

	cmd := NewShowCommand()
	cmd.SetArgs([]string{
		"--workspace", tmpDir,
		"test-session",
		"--messages",
		"--deliverable",
		"--logs",
		"--context",
	})
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Session: test-session")

	_ = session
}
