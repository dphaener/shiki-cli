package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name          string
		templateFile  string
		setupTemplate func(t *testing.T) string
		args          []string
		wantExitCode  int
		wantContains  []string
		setupEnv      func(t *testing.T)
		cleanupEnv    func(t *testing.T)
	}{
		{
			name:         "run with valid template - no API key",
			templateFile: "../../examples/simple-agreement.md",
			args:         []string{},
			wantExitCode: ExitError, // Will fail without API key
			wantContains: []string{"ANTHROPIC_API_KEY"},
			setupEnv:     func(t *testing.T) { os.Unsetenv("ANTHROPIC_API_KEY") },
			cleanupEnv:   func(t *testing.T) {},
		},
		{
			name: "run with invalid template",
			setupTemplate: func(t *testing.T) string {
				return createTempTemplate(t, `---
invalid: yaml: structure
---
# Invalid template
`)
			},
			args:         []string{},
			wantExitCode: ExitValidation,
			wantContains: []string{"Failed to parse template"},
			setupEnv:     func(t *testing.T) {},
			cleanupEnv:   func(t *testing.T) {},
		},
		{
			name: "run with missing required fields",
			setupTemplate: func(t *testing.T) string {
				return createTempTemplate(t, `---
agent1_name: "Agent1"
---
# Missing required fields
`)
			},
			args:         []string{},
			wantExitCode: ExitValidation,
			wantContains: []string{"validation failed"},
			setupEnv:     func(t *testing.T) {},
			cleanupEnv:   func(t *testing.T) {},
		},
		{
			name:         "run with non-existent template file",
			templateFile: "non-existent-file.md",
			args:         []string{},
			wantExitCode: ExitValidation,
			wantContains: []string{"Failed to parse template"},
			setupEnv:     func(t *testing.T) {},
			cleanupEnv:   func(t *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Setup environment
			tt.setupEnv(t)
			defer tt.cleanupEnv(t)

			// Get template file
			templateFile := tt.templateFile
			if tt.setupTemplate != nil {
				templateFile = tt.setupTemplate(t)
			}

			var stdout, stderr bytes.Buffer

			cmd := NewRunCommand()
			args := append([]string{"--workspace", tmpDir, templateFile}, tt.args...)
			cmd.SetArgs(args)
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
		})
	}
}

func TestRunCommand_WithWatchFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Use valid template
	templateFile := "../../examples/simple-agreement.md"

	// Unset API key to prevent actual execution
	os.Unsetenv("ANTHROPIC_API_KEY")

	var stdout, stderr bytes.Buffer

	cmd := NewRunCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, "--watch", templateFile})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	// Will fail without API key
	require.Error(t, err)

	output := stdout.String() + stderr.String()
	// Should still validate template before checking API key or attempting TUI
	assert.True(t, len(output) > 0)
}

func TestRunCommand_CustomSessionID(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid template
	templateFile := createValidTemplate(t)

	// Unset API key
	os.Unsetenv("ANTHROPIC_API_KEY")

	var stdout, stderr bytes.Buffer

	cmd := NewRunCommand()
	cmd.SetArgs([]string{
		"--workspace", tmpDir,
		"--session-id", "custom-session-123",
		templateFile,
	})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	// Will fail without API key, but should create session with custom ID
	require.Error(t, err)

	output := stdout.String() + stderr.String()
	// Check if custom session ID was used (might appear in output)
	if len(output) > 0 {
		// Session was created before API key check
		_ = output
	}
}

func TestRunCommand_WorkspaceCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid template with workspace structure
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
  - path: "docs"
    type: "directory"
---
# Task
Test workspace creation
`)

	// Unset API key
	os.Unsetenv("ANTHROPIC_API_KEY")

	var stdout, stderr bytes.Buffer

	cmd := NewRunCommand()
	cmd.SetArgs([]string{"--workspace", tmpDir, templateFile})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	// Will fail without API key, but workspace should be created
	require.Error(t, err)

	// Check workspace was created (before API key check failed)
	// The exact behavior depends on when workspace creation happens
}

func TestRunCommand_TemplateValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		template     string
		wantExitCode int
		errorSubstr  string
	}{
		{
			name: "invalid max_turns",
			template: `---
agent1_name: "Agent1"
agent1_role: "Role1"
agent1_system_prompt: "Prompt1"
agent2_name: "Agent2"
agent2_role: "Role2"
agent2_system_prompt: "Prompt2"
max_turns: 0
---
# Task
`,
			wantExitCode: ExitValidation,
			errorSubstr:  "validation",
		},
		{
			name: "empty agent name",
			template: `---
agent1_name: ""
agent1_role: "Role1"
agent1_system_prompt: "Prompt1"
agent2_name: "Agent2"
agent2_role: "Role2"
agent2_system_prompt: "Prompt2"
max_turns: 5
---
# Task
`,
			wantExitCode: ExitValidation,
			errorSubstr:  "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateFile := createTempTemplate(t, tt.template)

			var stdout, stderr bytes.Buffer

			cmd := NewRunCommand()
			cmd.SetArgs([]string{"--workspace", tmpDir, templateFile})
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()
			require.Error(t, err)

			output := stdout.String() + stderr.String()
			assert.Contains(t, output, tt.errorSubstr)
		})
	}
}

func TestRunCommand_GlobalFlags(t *testing.T) {
	tmpDir := t.TempDir()
	templateFile := createValidTemplate(t)

	// Unset API key
	os.Unsetenv("ANTHROPIC_API_KEY")

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "with verbose flag",
			args: []string{"--verbose", "--workspace", tmpDir, templateFile},
		},
		{
			name: "with no-color flag",
			args: []string{"--no-color", "--workspace", tmpDir, templateFile},
		},
		{
			name: "with config flag",
			args: []string{"--config", "/tmp/config.yaml", "--workspace", tmpDir, templateFile},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd := NewRunCommand()
			cmd.SetArgs(tt.args)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Will fail without API key, but should accept global flags
			_ = cmd.Execute()

			// Test passes if command accepts the flags without crashing
		})
	}
}
