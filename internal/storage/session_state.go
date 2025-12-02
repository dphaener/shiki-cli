package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dphaener/shiki-cli/pkg/types"
)

// SessionState wraps Session for JSON persistence
type SessionState struct {
	Version string         `json:"version"`
	Session *types.Session `json:"session"`
}

// SaveSessionState writes session to session_state.json
func SaveSessionState(session *types.Session) error {
	state := SessionState{
		Version: "1.0",
		Session: session,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}

	path := filepath.Join(session.WorkspaceDir, "session_state.json")
	if err := AtomicWrite(path, data, 0o600); err != nil {
		return fmt.Errorf("write session state: %w", err)
	}

	return nil
}

// LoadSessionState reads session from session_state.json
func LoadSessionState(workspaceDir string) (*types.Session, error) {
	path := filepath.Join(workspaceDir, "session_state.json")

	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		return nil, fmt.Errorf("read session state: %w", err)
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal session state: %w", err)
	}

	// Validate version
	if state.Version != "1.0" {
		return nil, fmt.Errorf("unsupported session state version: %s", state.Version)
	}

	return state.Session, nil
}
