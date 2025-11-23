package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darinhaener/collab/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldWorkspace(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt 1",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt 2",
		MaxTurns:           10,
		TaskBody:           "# Task\n\nDesign a system.",
	}

	baseDir := t.TempDir()
	sessionID := "test-session"

	wsDir, err := ScaffoldWorkspace(sessionID, baseDir, tmpl)
	require.NoError(t, err)

	// Verify workspace created
	assert.DirExists(t, wsDir)
	assert.Equal(t, filepath.Join(baseDir, sessionID), wsDir)

	// Verify standard files exist
	assert.FileExists(t, filepath.Join(wsDir, "task.md"))
	assert.FileExists(t, filepath.Join(wsDir, "shared_context.md"))
	assert.DirExists(t, filepath.Join(wsDir, "messages"))
	assert.DirExists(t, filepath.Join(wsDir, "memory"))

	// Verify task.md content
	content, err := os.ReadFile(filepath.Join(wsDir, "task.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "agent1_name: Alice")
	assert.Contains(t, string(content), "# Task")
	assert.Contains(t, string(content), "Design a system.")
}

func TestScaffoldWorkspaceWithStructure(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt 1",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt 2",
		MaxTurns:           10,
		TaskBody:           "# Task\n\nDesign a system.",
		WorkspaceStructure: []storage.WorkspaceStructure{
			{Path: "src/", Type: "directory"},
			{Path: "tests/", Type: "directory"},
			{Path: "README.md", Type: "file", Content: "# Project\n"},
			{Path: "src/main.go", Type: "file", Content: "package main\n"},
		},
	}

	baseDir := t.TempDir()
	sessionID := "test-session"

	wsDir, err := ScaffoldWorkspace(sessionID, baseDir, tmpl)
	require.NoError(t, err)

	// Verify custom directories
	assert.DirExists(t, filepath.Join(wsDir, "src"))
	assert.DirExists(t, filepath.Join(wsDir, "tests"))

	// Verify custom files
	assert.FileExists(t, filepath.Join(wsDir, "README.md"))
	assert.FileExists(t, filepath.Join(wsDir, "src/main.go"))

	// Verify file content
	content, err := os.ReadFile(filepath.Join(wsDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Project\n", string(content))

	content, err = os.ReadFile(filepath.Join(wsDir, "src/main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main\n", string(content))
}

func TestScaffoldWorkspaceNestedStructure(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt 1",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt 2",
		MaxTurns:           10,
		TaskBody:           "# Task",
		WorkspaceStructure: []storage.WorkspaceStructure{
			{Path: "designs/architecture.md", Type: "file", Content: "# Architecture\n"},
			{Path: "designs/data-model.md", Type: "file", Content: "# Data Model\n"},
		},
	}

	baseDir := t.TempDir()
	sessionID := "test-session"

	wsDir, err := ScaffoldWorkspace(sessionID, baseDir, tmpl)
	require.NoError(t, err)

	// Verify nested files
	assert.FileExists(t, filepath.Join(wsDir, "designs/architecture.md"))
	assert.FileExists(t, filepath.Join(wsDir, "designs/data-model.md"))

	// Verify parent directory created
	assert.DirExists(t, filepath.Join(wsDir, "designs"))
}
