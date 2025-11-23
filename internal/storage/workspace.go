package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// WorkspaceStructure defines initial workspace files/dirs
type WorkspaceStructure struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"` // "file" or "directory"
	Content string `yaml:"content,omitempty"`
}

// CreateWorkspace initializes session directory
func CreateWorkspace(sessionID, baseDir string, structure []WorkspaceStructure) (string, error) {
	workspaceDir := filepath.Join(baseDir, sessionID)

	// Create root with restricted permissions (SEC-002)
	if err := os.MkdirAll(workspaceDir, 0o700); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}

	// Create standard directories
	standardDirs := []string{"messages", "memory"}
	for _, dir := range standardDirs {
		path := filepath.Join(workspaceDir, dir)
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", fmt.Errorf("create %s dir: %w", dir, err)
		}
	}

	// Create initial files
	initialFiles := map[string]string{
		"shared_context.md":        "# Shared Context\n\n",
		"memory/agent_1_memory.md": "# Agent 1 Memory\n\n",
		"memory/agent_2_memory.md": "# Agent 2 Memory\n\n",
	}

	for relPath, content := range initialFiles {
		path := filepath.Join(workspaceDir, relPath)
		if err := AtomicWriteString(path, content, 0o600); err != nil {
			return "", fmt.Errorf("create %s: %w", relPath, err)
		}
	}

	// Apply template structure if provided
	for _, item := range structure {
		path := filepath.Join(workspaceDir, item.Path)

		if item.Type == "directory" {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return "", fmt.Errorf("create custom dir %s: %w", item.Path, err)
			}
		} else if item.Type == "file" {
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return "", fmt.Errorf("create parent dir for %s: %w", item.Path, err)
			}
			content := item.Content
			if content == "" {
				content = ""
			}
			if err := AtomicWriteString(path, content, 0o600); err != nil {
				return "", fmt.Errorf("create custom file %s: %w", item.Path, err)
			}
		}
	}

	return workspaceDir, nil
}

// CleanupWorkspace removes session directory
func CleanupWorkspace(workspaceDir string) error {
	return os.RemoveAll(workspaceDir)
}

// WorkspaceExists checks if session workspace exists
func WorkspaceExists(sessionID, baseDir string) bool {
	path := filepath.Join(baseDir, sessionID)
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
