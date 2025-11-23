package storage

import (
	"os"
	"path/filepath"
)

// WriteSharedContext writes the shared context file
func WriteSharedContext(workspaceDir, content string) error {
	path := filepath.Join(workspaceDir, "shared_context.md")
	return AtomicWriteString(path, content, 0600)
}

// ReadSharedContext reads the shared context file
func ReadSharedContext(workspaceDir string) (string, error) {
	path := filepath.Join(workspaceDir, "shared_context.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Return empty if doesn't exist yet
		}
		return "", err
	}
	return string(data), nil
}
