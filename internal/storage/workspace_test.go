package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWorkspace(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "20251123-120000-test"

	structure := []WorkspaceStructure{
		{Path: "designs", Type: "directory"},
		{Path: "designs/draft.md", Type: "file", Content: "# Draft\n"},
	}

	wsDir, err := CreateWorkspace(sessionID, baseDir, structure)
	require.NoError(t, err)

	// Verify standard structure
	assert.DirExists(t, filepath.Join(wsDir, "messages"))
	assert.DirExists(t, filepath.Join(wsDir, "memory"))
	assert.FileExists(t, filepath.Join(wsDir, "shared_context.md"))
	assert.FileExists(t, filepath.Join(wsDir, "memory", "agent_1_memory.md"))
	assert.FileExists(t, filepath.Join(wsDir, "memory", "agent_2_memory.md"))

	// Verify custom structure
	assert.DirExists(t, filepath.Join(wsDir, "designs"))
	assert.FileExists(t, filepath.Join(wsDir, "designs", "draft.md"))

	content, _ := os.ReadFile(filepath.Join(wsDir, "designs", "draft.md")) //nolint:gosec // G304: Test file path
	assert.Equal(t, "# Draft\n", string(content))
}

func TestCreateWorkspacePermissions(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "20251123-120000-perms"

	wsDir, err := CreateWorkspace(sessionID, baseDir, nil)
	require.NoError(t, err)

	// Check workspace dir permissions
	info, err := os.Stat(wsDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())

	// Check file permissions
	info, err = os.Stat(filepath.Join(wsDir, "shared_context.md"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestWorkspaceExists(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "20251123-120000-exists"

	// Should not exist initially
	assert.False(t, WorkspaceExists(sessionID, baseDir))

	// Create workspace
	_, err := CreateWorkspace(sessionID, baseDir, nil)
	require.NoError(t, err)

	// Should exist now
	assert.True(t, WorkspaceExists(sessionID, baseDir))
}

func TestCleanupWorkspace(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "20251123-120000-cleanup"

	wsDir, err := CreateWorkspace(sessionID, baseDir, nil)
	require.NoError(t, err)

	// Verify it exists
	assert.DirExists(t, wsDir)

	// Cleanup
	err = CleanupWorkspace(wsDir)
	require.NoError(t, err)

	// Verify it's gone
	assert.NoDirExists(t, wsDir)
}

func TestCreateWorkspaceNestedStructure(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "20251123-120000-nested"

	structure := []WorkspaceStructure{
		{Path: "src", Type: "directory"},
		{Path: "src/components", Type: "directory"},
		{Path: "src/components/app.go", Type: "file", Content: "package main\n"},
		{Path: "README.md", Type: "file", Content: "# Project\n"},
	}

	wsDir, err := CreateWorkspace(sessionID, baseDir, structure)
	require.NoError(t, err)

	// Verify nested structure
	assert.DirExists(t, filepath.Join(wsDir, "src"))
	assert.DirExists(t, filepath.Join(wsDir, "src", "components"))
	assert.FileExists(t, filepath.Join(wsDir, "src", "components", "app.go"))
	assert.FileExists(t, filepath.Join(wsDir, "README.md"))

	content, _ := os.ReadFile(filepath.Join(wsDir, "src", "components", "app.go")) //nolint:gosec // G304: Test file path
	assert.Equal(t, "package main\n", string(content))
}
