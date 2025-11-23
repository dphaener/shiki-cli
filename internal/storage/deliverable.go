package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DeliverableMetadata holds metadata for deliverable submissions
type DeliverableMetadata struct {
	SubmittedBy string    `yaml:"submitted_by"`
	SubmittedAt time.Time `yaml:"submitted_at"`
	ApprovedBy  []string  `yaml:"approved_by"`
}

// WriteDeliverable writes the deliverable file with metadata
func WriteDeliverable(workspaceDir, content string, metadata DeliverableMetadata) error {
	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal deliverable metadata: %w", err)
	}

	fullContent := fmt.Sprintf("---\n%s---\n\n%s", string(yamlData), content)
	path := filepath.Join(workspaceDir, "deliverable.md")
	return AtomicWriteString(path, fullContent, 0o600)
}

// ReadDeliverable reads the deliverable file and its metadata
func ReadDeliverable(workspaceDir string) (string, *DeliverableMetadata, error) {
	path := filepath.Join(workspaceDir, "deliverable.md")
	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		return "", nil, err
	}

	// Split frontmatter and content
	parts := strings.SplitN(string(data), "---\n", 3)
	if len(parts) < 3 {
		// No metadata, return content only
		return string(data), nil, nil
	}

	var meta DeliverableMetadata
	if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return "", nil, fmt.Errorf("parse deliverable metadata: %w", err)
	}

	return strings.TrimSpace(parts[2]), &meta, nil
}
