package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name           string
		templateFile   string
		wantExitCode   int
		wantOutputsubs string
	}{
		{
			name:           "valid template",
			templateFile:   "../../examples/simple-agreement.md",
			wantExitCode:   ExitSuccess,
			wantOutputsubs: "Template is valid",
		},
		{
			name:           "non-existent file",
			templateFile:   "non-existent-file.md",
			wantExitCode:   ExitValidation,
			wantOutputsubs: "Template file not found",
		},
		{
			name: "invalid YAML",
			templateFile: createTempTemplate(t, `---
invalid yaml: [unclosed array
---
# Task
Test task
`),
			wantExitCode:   ExitValidation,
			wantOutputsubs: "Failed to parse template",
		},
		{
			name: "missing required fields",
			templateFile: createTempTemplate(t, `---
agent1_name: "Agent1"
---
# Task
Missing required fields
`),
			wantExitCode:   ExitValidation,
			wantOutputsubs: "Template validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture actual stderr since that's where output goes
			oldStderr := os.Stderr
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stderr = w
			os.Stdout = w

			// Create command
			cmd := NewValidateCommand()
			cmd.SetArgs([]string{tt.templateFile})

			// Execute command
			err := cmd.Execute()

			// Restore and capture output
			w.Close()
			os.Stderr = oldStderr
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check exit code
			if tt.wantExitCode == ExitSuccess {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				exitErr, ok := err.(*exitCodeError)
				require.True(t, ok, "expected exitCodeError, got %T", err)
				assert.Equal(t, tt.wantExitCode, exitErr.code)
			}

			// Check output contains expected substring
			assert.Contains(t, output, tt.wantOutputsubs)
		})
	}
}

func TestValidateCommand_DisplaysAgentInfo(t *testing.T) {
	// Use the simple-agreement example template
	templateFile := "../../examples/simple-agreement.md"

	// Capture actual output
	oldStderr := os.Stderr
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stderr = w
	os.Stdout = w

	cmd := NewValidateCommand()
	cmd.SetArgs([]string{templateFile})

	err := cmd.Execute()

	// Restore and capture
	w.Close()
	os.Stderr = oldStderr
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)

	// Check that agent information is displayed
	assert.Contains(t, output, "Agent 1:")
	assert.Contains(t, output, "Agent 2:")
	assert.Contains(t, output, "Max turns:")
}

func TestValidateCommand_WorkspaceStructure(t *testing.T) {
	// Create template with workspace structure
	templateFile := createTempTemplate(t, `---
agent1_name: "Agent1"
agent1_role: "Developer"
agent1_system_prompt: "You are a developer"
agent2_name: "Agent2"
agent2_role: "Reviewer"
agent2_system_prompt: "You are a reviewer"
max_turns: 5
workspace_structure:
  - path: "src"
    type: "directory"
  - path: "src/main.go"
    type: "file"
    content: "package main"
---
# Task
Test with workspace structure
`)

	// Capture actual output
	oldStderr := os.Stderr
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stderr = w
	os.Stdout = w

	cmd := NewValidateCommand()
	cmd.SetArgs([]string{templateFile})

	err := cmd.Execute()

	// Restore and capture
	w.Close()
	os.Stderr = oldStderr
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "Workspace structure: 2 items defined")
}
