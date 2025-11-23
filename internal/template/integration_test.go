package template

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateIntegration(t *testing.T) {
	// Parse example template
	tmpl, err := ParseTemplate("../../examples/simple-agreement.md")
	require.NoError(t, err)

	// Validate
	err = ValidateTemplate(tmpl)
	require.NoError(t, err)

	// Scaffold workspace
	baseDir := t.TempDir()
	wsDir, err := ScaffoldWorkspace("test-session", baseDir, tmpl)
	require.NoError(t, err)

	// Verify workspace structure
	assert.DirExists(t, wsDir)
	assert.FileExists(t, filepath.Join(wsDir, "task.md"))
	assert.FileExists(t, filepath.Join(wsDir, "shared_context.md"))
}

func TestAllExamplesValid(t *testing.T) {
	examples := []string{
		"../../examples/simple-agreement.md",
		"../../examples/code-review.md",
		"../../examples/architecture-design.md",
	}

	for _, examplePath := range examples {
		t.Run(filepath.Base(examplePath), func(t *testing.T) {
			// Parse
			tmpl, err := ParseTemplate(examplePath)
			require.NoError(t, err, "Failed to parse %s", examplePath)

			// Validate
			err = ValidateTemplate(tmpl)
			require.NoError(t, err, "Failed to validate %s", examplePath)

			// Basic structure checks
			assert.NotEmpty(t, tmpl.Agent1Name)
			assert.NotEmpty(t, tmpl.Agent2Name)
			assert.Greater(t, tmpl.MaxTurns, 0)
			assert.NotEmpty(t, tmpl.TaskBody)
		})
	}
}

func TestExamplesProgression(t *testing.T) {
	// Simple example
	simple, err := ParseTemplate("../../examples/simple-agreement.md")
	require.NoError(t, err)
	assert.Len(t, simple.WorkspaceStructure, 0, "Simple example should have no workspace structure")
	assert.Nil(t, simple.CostLimits, "Simple example should have no cost limits")

	// Medium complexity example
	codeReview, err := ParseTemplate("../../examples/code-review.md")
	require.NoError(t, err)
	assert.Greater(t, len(codeReview.WorkspaceStructure), 0, "Code review example should have workspace structure")
	assert.NotNil(t, codeReview.CostLimits, "Code review example should have cost limits")

	// Complex example
	architecture, err := ParseTemplate("../../examples/architecture-design.md")
	require.NoError(t, err)
	assert.Greater(t, len(architecture.WorkspaceStructure), 0, "Architecture example should have workspace structure")
	assert.NotNil(t, architecture.CostLimits, "Architecture example should have cost limits")
	assert.NotEmpty(t, architecture.CompletionCriteria, "Architecture example should have completion criteria")

	// Verify progression in max_turns
	assert.Less(t, simple.MaxTurns, codeReview.MaxTurns, "Code review should have more turns than simple")
	assert.Less(t, codeReview.MaxTurns, architecture.MaxTurns, "Architecture should have more turns than code review")
}
