package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanCommand(t *testing.T) {
	tests := []struct {
		name          string
		setupSessions func(t *testing.T, tmpDir string) []string
		args          []string
		wantExitCode  int
		wantRemaining int
		wantContains  []string
	}{
		{
			name: "clean all sessions with force",
			setupSessions: func(t *testing.T, tmpDir string) []string {
				s1 := createTestSession(t, tmpDir, "session-1", types.SessionCompleted)
				s2 := createTestSession(t, tmpDir, "session-2", types.SessionCompleted)
				return []string{s1.ID, s2.ID}
			},
			args:          []string{"--force"},
			wantExitCode:  ExitSuccess,
			wantRemaining: 0,
			wantContains:  []string{"Cleaned"},
		},
		{
			name: "clean completed sessions only",
			setupSessions: func(t *testing.T, tmpDir string) []string {
				s1 := createTestSession(t, tmpDir, "completed-1", types.SessionCompleted)
				s2 := createTestSession(t, tmpDir, "running-1", types.SessionRunning)
				return []string{s1.ID, s2.ID}
			},
			args:          []string{"--status", "completed", "--force"},
			wantExitCode:  ExitSuccess,
			wantRemaining: 1, // running session should remain
			wantContains:  []string{"Cleaned"},
		},
		{
			name: "clean with age filter",
			setupSessions: func(t *testing.T, tmpDir string) []string {
				// Create old session (7 days ago)
				oldSession := createTestSession(t, tmpDir, "old-session", types.SessionCompleted)
				oldTime := time.Now().Add(-7 * 24 * time.Hour)
				oldSession.StartedAt = &oldTime

				// Create recent session (1 day ago)
				recentSession := createTestSession(t, tmpDir, "recent-session", types.SessionCompleted)

				return []string{oldSession.ID, recentSession.ID}
			},
			args:          []string{"--age", "3", "--force"}, // Clean sessions older than 3 days
			wantExitCode:  ExitSuccess,
			wantRemaining: 1, // recent session should remain
			wantContains:  []string{"Cleaned"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Setup sessions
			sessionIDs := tt.setupSessions(t, tmpDir)

			var stdout bytes.Buffer

			cmd := NewCleanCommand()
			args := append([]string{"--workspace", tmpDir}, tt.args...)
			cmd.SetArgs(args)
			cmd.SetOut(&stdout)

			err := cmd.Execute()

			if tt.wantExitCode == ExitSuccess {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			output := stdout.String()

			// Check output contains expected substrings
			for _, substr := range tt.wantContains {
				assert.Contains(t, output, substr)
			}

			// Check remaining sessions
			entries, err := os.ReadDir(tmpDir)
			require.NoError(t, err)

			remainingCount := 0
			for _, entry := range entries {
				if entry.IsDir() {
					remainingCount++
				}
			}

			assert.Equal(t, tt.wantRemaining, remainingCount,
				"expected %d remaining sessions, got %d", tt.wantRemaining, remainingCount)

			_ = sessionIDs
		})
	}
}

func TestCleanCommand_WithoutForce(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test session
	_ = createTestSession(t, tmpDir, "test-session", types.SessionCompleted)

	var stdout bytes.Buffer

	cmd := NewCleanCommand()
	// Simulate user declining confirmation by providing "n" input
	cmd.SetArgs([]string{"--workspace", tmpDir})
	cmd.SetOut(&stdout)
	cmd.SetIn(strings.NewReader("n\n"))

	err := cmd.Execute()

	// Without --force, command should ask for confirmation
	// If user says "n", nothing should be deleted
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	sessionDirs := 0
	for _, entry := range entries {
		if entry.IsDir() {
			sessionDirs++
		}
	}

	// Session should still exist
	assert.Equal(t, 1, sessionDirs)
}

func TestCleanCommand_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout bytes.Buffer

	cmd := NewCleanCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "--force"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	// Should handle empty workspace gracefully
	assert.Contains(t, output, "0 session")
}

func TestCleanCommand_NonExistentWorkspace(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")

	var stdout, stderr bytes.Buffer

	cmd := NewCleanCommand()
	cmd.SetArgs([]string{"--workspace", nonExistentDir, "--force"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	// Should handle non-existent workspace gracefully
	if err != nil {
		// Either error or handle gracefully
		output := stdout.String() + stderr.String()
		assert.True(t, len(output) > 0)
	}
}

func TestCleanCommand_StatusFilters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create sessions with different statuses
	_ = createTestSession(t, tmpDir, "completed-1", types.SessionCompleted)
	_ = createTestSession(t, tmpDir, "incomplete-1", types.SessionIncomplete)
	_ = createTestSession(t, tmpDir, "paused-1", types.SessionPaused)
	_ = createTestSession(t, tmpDir, "error-1", types.SessionError)

	statusFilters := []struct {
		status        string
		wantRemaining int
	}{
		{"completed", 3},  // Remove completed, leave 3
		{"incomplete", 3}, // Remove incomplete, leave 3
		{"paused", 3},     // Remove paused, leave 3
		{"error", 3},      // Remove error, leave 3
	}

	for _, sf := range statusFilters {
		t.Run("filter_"+sf.status, func(t *testing.T) {
			// Use a fresh tmpDir for each subtest
			testDir := t.TempDir()

			// Recreate sessions
			_ = createTestSession(t, testDir, "completed-1", types.SessionCompleted)
			_ = createTestSession(t, testDir, "incomplete-1", types.SessionIncomplete)
			_ = createTestSession(t, testDir, "paused-1", types.SessionPaused)
			_ = createTestSession(t, testDir, "error-1", types.SessionError)

			var stdout bytes.Buffer

			cmd := NewCleanCommand()
			cmd.SetArgs([]string{"--workspace", testDir, "--status", sf.status, "--force"})
			cmd.SetOut(&stdout)

			err := cmd.Execute()
			require.NoError(t, err)

			// Check remaining sessions
			entries, err := os.ReadDir(testDir)
			require.NoError(t, err)

			remainingCount := 0
			for _, entry := range entries {
				if entry.IsDir() {
					remainingCount++
				}
			}

			assert.Equal(t, sf.wantRemaining, remainingCount)
		})
	}
}
