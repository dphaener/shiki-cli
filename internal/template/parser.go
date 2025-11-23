package template

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseTemplate reads and parses a task template file
func ParseTemplate(filePath string) (*TaskTemplate, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read template file: %w", err)
	}

	return ParseTemplateBytes(data)
}

// ParseTemplateBytes parses template from byte slice
func ParseTemplateBytes(data []byte) (*TaskTemplate, error) {
	content := string(data)

	// Split frontmatter and body
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid template format: missing YAML frontmatter (expected ---\\n...\\n---\\n)")
	}

	frontmatter := parts[1]
	body := strings.TrimSpace(parts[2])

	// Start with defaults
	tmpl := Defaults()

	// Parse YAML frontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), tmpl); err != nil {
		// Extract line number from YAML error for helpful message
		return nil, formatYAMLError(err, frontmatter)
	}

	tmpl.TaskBody = body

	return tmpl, nil
}

func formatYAMLError(err error, frontmatter string) error {
	errMsg := err.Error()

	// Try to extract line number from yaml.v3 error
	// Format: "yaml: line X: ..."
	if strings.Contains(errMsg, "line ") {
		return fmt.Errorf("YAML frontmatter error: %s", errMsg)
	}

	// Count lines up to error position if available
	lines := strings.Split(frontmatter, "\n")
	return fmt.Errorf("YAML frontmatter error (total %d lines): %s", len(lines), errMsg)
}
