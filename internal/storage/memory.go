package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteMemory writes an agent's memory file
func WriteMemory(workspaceDir, agentID, content string) error {
	filename := fmt.Sprintf("%s_memory.md", agentID)
	path := filepath.Join(workspaceDir, "memory", filename)
	return AtomicWriteString(path, content, 0o600)
}

// ReadMemory reads an agent's memory file
func ReadMemory(workspaceDir, agentID string) (string, error) {
	filename := fmt.Sprintf("%s_memory.md", agentID)
	path := filepath.Join(workspaceDir, "memory", filename)
	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
