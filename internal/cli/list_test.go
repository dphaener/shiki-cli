package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCommand(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create test sessions
	session1 := createTestSession(t, tmpDir, "session-1", types.SessionCompleted)
	session2 := createTestSession(t, tmpDir, "session-2", types.SessionPaused)
	session3 := createTestSession(t, tmpDir, "session-3", types.SessionRunning)

	tests := []struct {
		name           string
		args           []string
		wantSessions   []string
		wantExitCode   int
		outputContains []string
	}{
		{
			name:         "list all sessions - table format",
			args:         []string{"--workspace", tmpDir, "--format", "table"},
			wantSessions: []string{"session-1", "session-2", "session-3"},
			wantExitCode: ExitSuccess,
			outputContains: []string{
				"SESSION ID",
				"STATUS",
				"TURNS",
				"Total: 3 session(s)",
			},
		},
		{
			name:         "filter by status - completed",
			args:         []string{"--workspace", tmpDir, "--status", "completed", "--format", "table"},
			wantSessions: []string{"session-1"},
			wantExitCode: ExitSuccess,
			outputContains: []string{
				"session-1",
				"Total: 1 session(s)",
			},
		},
		{
			name:         "filter by status - paused",
			args:         []string{"--workspace", tmpDir, "--status", "paused", "--format", "table"},
			wantSessions: []string{"session-2"},
			wantExitCode: ExitSuccess,
			outputContains: []string{
				"session-2",
				"Total: 1 session(s)",
			},
		},
		{
			name:         "JSON output format",
			args:         []string{"--workspace", tmpDir, "--format", "json"},
			wantSessions: []string{"session-1", "session-2", "session-3"},
			wantExitCode: ExitSuccess,
			outputContains: []string{
				`"sessions"`,
				`"total": 3`,
				`"id": "session-1"`,
			},
		},
		{
			name:         "CSV output format",
			args:         []string{"--workspace", tmpDir, "--format", "csv"},
			wantSessions: []string{"session-1", "session-2", "session-3"},
			wantExitCode: ExitSuccess,
			outputContains: []string{
				"session_id,status,task_name",
				"session-1,completed",
				"session-2,paused",
				"session-3,running",
			},
		},
		{
			name:         "limit sessions",
			args:         []string{"--workspace", tmpDir, "--format", "table", "--limit", "2"},
			wantExitCode: ExitSuccess,
			outputContains: []string{
				"Total: 2 session(s)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			cmd := NewListCommand()
			cmd.SetArgs(tt.args)
			cmd.SetOut(&stdout)

			err := cmd.Execute()

			if tt.wantExitCode == ExitSuccess {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			output := stdout.String()

			// Check output contains expected substrings
			for _, substr := range tt.outputContains {
				assert.Contains(t, output, substr, "output should contain: %s", substr)
			}

			// For session filtering tests, ensure unwanted sessions are not present
			if tt.name == "filter by status - completed" {
				assert.NotContains(t, output, "session-2")
				assert.NotContains(t, output, "session-3")
			} else if tt.name == "filter by status - paused" {
				assert.NotContains(t, output, "session-1")
				assert.NotContains(t, output, "session-3")
			}
		})
	}

	// Cleanup
	_ = session1
	_ = session2
	_ = session3
}

func TestListCommand_EmptyWorkspace(t *testing.T) {
	// Create empty workspace
	tmpDir := t.TempDir()

	var stdout bytes.Buffer

	cmd := NewListCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "--format", "table"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "No sessions found")
}

func TestListCommand_NonExistentWorkspace(t *testing.T) {
	// Use non-existent directory
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")

	var stdout bytes.Buffer

	cmd := NewListCommand()
	cmd.SetArgs([]string{"--workspace", nonExistentDir, "--format", "table"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "No sessions found")
}

func TestListCommand_JSONOutputStructure(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create one test session
	_ = createTestSession(t, tmpDir, "test-session", types.SessionCompleted)

	var stdout bytes.Buffer

	cmd := NewListCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "--format", "json"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse JSON output
	var result struct {
		Sessions []struct {
			ID       string `json:"id"`
			Status   string `json:"status"`
			TaskName string `json:"task_name"`
			Turns    struct {
				Current int `json:"current"`
				Max     int `json:"max"`
			} `json:"turns"`
			Duration float64 `json:"duration_seconds"`
			Cost     struct {
				Total  float64 `json:"total"`
				Agent1 float64 `json:"agent1"`
				Agent2 float64 `json:"agent2"`
			} `json:"cost"`
			StartedAt   string  `json:"started_at"`
			CompletedAt *string `json:"completed_at,omitempty"`
		} `json:"sessions"`
		Total int `json:"total"`
	}

	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)

	// Validate JSON structure
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Sessions, 1)
	assert.Equal(t, "test-session", result.Sessions[0].ID)
	assert.Equal(t, "completed", result.Sessions[0].Status)
}

func TestListCommand_CSVOutputStructure(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Create one test session
	_ = createTestSession(t, tmpDir, "test-session", types.SessionCompleted)

	var stdout bytes.Buffer

	cmd := NewListCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "--format", "csv"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse CSV output
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2) // header + 1 session

	// Check header
	header := lines[0]
	assert.Contains(t, header, "session_id")
	assert.Contains(t, header, "status")
	assert.Contains(t, header, "task_name")
	assert.Contains(t, header, "current_turn")
	assert.Contains(t, header, "max_turns")
	assert.Contains(t, header, "duration_seconds")
	assert.Contains(t, header, "total_cost")

	// Check data row
	dataRow := lines[1]
	assert.Contains(t, dataRow, "test-session")
	assert.Contains(t, dataRow, "completed")
}

func TestListCommand_InvalidOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	cmd := NewListCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "--format", "invalid"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.Error(t, err)

	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Invalid output format")
}
