package template

import (
	"fmt"
	"path/filepath"

	"github.com/darinhaener/collab/internal/storage"
	"gopkg.in/yaml.v3"
)

// ScaffoldWorkspace creates workspace from template structure
func ScaffoldWorkspace(sessionID, baseDir string, tmpl *TaskTemplate) (string, error) {
	// Use storage layer's CreateWorkspace
	workspaceDir, err := storage.CreateWorkspace(
		sessionID,
		baseDir,
		tmpl.WorkspaceStructure,
	)
	if err != nil {
		return "", fmt.Errorf("scaffold workspace: %w", err)
	}

	// Copy task template to workspace for reference
	templateContent := serializeTemplate(tmpl)
	if err := storage.AtomicWriteString(
		filepath.Join(workspaceDir, "task.md"),
		templateContent,
		0o600,
	); err != nil {
		return "", fmt.Errorf("write task template: %w", err)
	}

	return workspaceDir, nil
}

func serializeTemplate(tmpl *TaskTemplate) string {
	// Create a copy for serialization (exclude TaskBody from YAML)
	yamlData, _ := yaml.Marshal(tmpl)
	return fmt.Sprintf("---\n%s---\n\n%s", string(yamlData), tmpl.TaskBody)
}
