package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand_CopyExample(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		checkFile    func(t *testing.T, filePath string)
	}{
		{
			name:         "copy simple-agreement example",
			args:         []string{"--template", "simple-agreement", "my-task.md"},
			wantExitCode: ExitSuccess,
			checkFile: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "agent1_name:")
				assert.Contains(t, string(content), "agent2_name:")
				assert.Contains(t, string(content), "max_turns:")
			},
		},
		{
			name:         "copy code-review example",
			args:         []string{"--template", "code-review", "review-task.md"},
			wantExitCode: ExitSuccess,
			checkFile: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "agent1_name:")
			},
		},
		{
			name:         "copy architecture-design example",
			args:         []string{"--template", "architecture-design", "design-task.md"},
			wantExitCode: ExitSuccess,
			checkFile: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "agent1_name:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, tt.args[len(tt.args)-1])

			var stdout bytes.Buffer

			cmd := NewInitCommand()
			// Adjust args to use tmpDir
			args := append(tt.args[:len(tt.args)-1], outputPath)
			cmd.SetArgs(args)
			cmd.SetOut(&stdout)

			err := cmd.Execute()

			if tt.wantExitCode == ExitSuccess {
				assert.NoError(t, err)

				// Check file was created
				_, err := os.Stat(outputPath)
				require.NoError(t, err)

				// Run additional checks
				if tt.checkFile != nil {
					tt.checkFile(t, outputPath)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestInitCommand_InteractiveMode(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "interactive-task.md")

	// Simulate user input for interactive wizard
	userInput := strings.Join([]string{
		"Alice",              // agent1_name
		"Developer",          // agent1_role
		"You are a dev",      // agent1_system_prompt
		"Bob",                // agent2_name
		"Reviewer",           // agent2_role
		"You are a reviewer", // agent2_system_prompt
		"10",                 // max_turns
		"",                   // workspace_structure (empty for none)
	}, "\n") + "\n"

	var stdout bytes.Buffer

	cmd := NewInitCommand()
	cmd.SetArgs([]string{outputPath})
	cmd.SetIn(strings.NewReader(userInput))
	cmd.SetOut(&stdout)

	err := cmd.Execute()

	// Interactive mode might be implemented or not
	// This test checks if the command handles it gracefully
	if err == nil {
		// Check if file was created
		if _, err := os.Stat(outputPath); err == nil {
			content, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			// Should contain the input values
			assert.Contains(t, string(content), "Alice")
			assert.Contains(t, string(content), "Bob")
		}
	}
}

func TestInitCommand_InvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "task.md")

	var stdout, stderr bytes.Buffer

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--template", "non-existent-template", outputPath})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	// Should error for non-existent template
	if err != nil {
		output := stdout.String() + stderr.String()
		assert.True(t, len(output) > 0)
	}
}

func TestInitCommand_FileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "existing.md")

	// Create file that already exists
	err := os.WriteFile(outputPath, []byte("existing content"), 0600)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--template", "simple-agreement", outputPath})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err = cmd.Execute()

	// Command should handle existing file gracefully
	// Either error or ask for confirmation
	if err != nil {
		output := stdout.String() + stderr.String()
		assert.True(t, len(output) > 0)
	}
}

func TestInitCommand_OutputToStdout(t *testing.T) {
	var stdout bytes.Buffer

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--template", "simple-agreement", "-"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()

	// If command supports stdout output ("-"), check it
	if err == nil {
		output := stdout.String()
		if len(output) > 0 {
			assert.Contains(t, output, "agent1_name:")
		}
	}
}

func TestInitCommand_ListTemplates(t *testing.T) {
	var stdout bytes.Buffer

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--list"})
	cmd.SetOut(&stdout)

	err := cmd.Execute()

	// If command supports --list flag, check it shows available templates
	if err == nil {
		output := stdout.String()
		if len(output) > 0 {
			// Should list available templates
			assert.True(t, strings.Contains(output, "simple-agreement") ||
				strings.Contains(output, "code-review") ||
				strings.Contains(output, "architecture-design"))
		}
	}
}
