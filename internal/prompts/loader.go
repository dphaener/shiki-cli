package prompts

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// SpecifyPromptData contains the data for rendering the specify prompt template
type SpecifyPromptData struct {
	FriendlyName   string
	FeatureNumber  int
	Slug           string
	Phase          string
	HasDescription bool
	FeatureDesc    string
	SpecFile       string
	SpecDir        string
	ChecklistDir   string
}

// LoadSpecifyPrompt loads and renders the specify system prompt
// It checks for an external override at .sekkei/templates/specify-prompt.md first,
// then falls back to the embedded default prompt
func LoadSpecifyPrompt(data SpecifyPromptData) (string, error) {
	promptContent := defaultSpecifyPrompt

	// Check for external override
	overridePath := filepath.Join(".sekkei", "templates", "specify-prompt.md")
	if content, err := os.ReadFile(overridePath); err == nil {
		promptContent = string(content)
	}

	// Parse and execute template
	tmpl, err := template.New("specify-prompt").Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("parse prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute prompt template: %w", err)
	}

	return buf.String(), nil
}

// LoadSpecifyPromptWithRepoRoot loads the prompt with a custom repo root path
func LoadSpecifyPromptWithRepoRoot(data SpecifyPromptData, repoRoot string) (string, error) {
	promptContent := defaultSpecifyPrompt

	// Check for external override relative to repo root
	overridePath := filepath.Join(repoRoot, ".sekkei", "templates", "specify-prompt.md")
	if content, err := os.ReadFile(overridePath); err == nil {
		promptContent = string(content)
	}

	// Parse and execute template
	tmpl, err := template.New("specify-prompt").Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("parse prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute prompt template: %w", err)
	}

	return buf.String(), nil
}

// PlanPromptData contains the data for rendering the plan prompt template
type PlanPromptData struct {
	FriendlyName  string
	FeatureNumber int
	SpecSlug      string
	Phase         string
	SpecFile      string
	PlanFile      string
	ContractsDir  string
	SpecDir       string
}

// LoadPlanPrompt loads and renders the plan system prompt
// It checks for an external override at .sekkei/templates/plan-prompt.md first,
// then falls back to the embedded default prompt
func LoadPlanPrompt(data PlanPromptData) (string, error) {
	promptContent := defaultPlanPrompt

	// Check for external override
	overridePath := filepath.Join(".sekkei", "templates", "plan-prompt.md")
	if content, err := os.ReadFile(overridePath); err == nil {
		promptContent = string(content)
	}

	// Parse and execute template
	tmpl, err := template.New("plan-prompt").Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("parse plan prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute plan prompt template: %w", err)
	}

	return buf.String(), nil
}
