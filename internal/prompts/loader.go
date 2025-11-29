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

// TasksPromptData contains the data for rendering the tasks prompt template
type TasksPromptData struct {
	FriendlyName  string
	FeatureNumber int
	Slug          string
	SpecFile      string
	PlanFile      string
	TasksFile     string
	FeatureDir    string
}

// LoadTasksPrompt loads and renders the tasks system prompt
// It checks for an external override at .sekkei/templates/tasks-prompt.md first,
// then falls back to the embedded default prompt
func LoadTasksPrompt(data TasksPromptData) (string, error) {
	promptContent := defaultTasksPrompt

	// Check for external override
	overridePath := filepath.Join(".sekkei", "templates", "tasks-prompt.md")
	if content, err := os.ReadFile(overridePath); err == nil {
		promptContent = string(content)
	}

	// Parse and execute template
	tmpl, err := template.New("tasks-prompt").Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("parse tasks prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute tasks prompt template: %w", err)
	}

	return buf.String(), nil
}

// ImplementPromptData contains the data for rendering the implement prompt template
type ImplementPromptData struct {
	FriendlyName  string
	FeatureNumber int
	Slug          string
	SpecFile      string
	PlanFile      string
	TasksFile     string
	FeatureDir    string
}

// LoadImplementPrompt loads and renders the implement system prompt
// It checks for an external override at .sekkei/templates/implement-prompt.md first,
// then falls back to the embedded default prompt
func LoadImplementPrompt(data ImplementPromptData) (string, error) {
	promptContent := defaultImplementPrompt

	// Check for external override
	overridePath := filepath.Join(".sekkei", "templates", "implement-prompt.md")
	if content, err := os.ReadFile(overridePath); err == nil {
		promptContent = string(content)
	}

	// Parse and execute template
	tmpl, err := template.New("implement-prompt").Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("parse implement prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute implement prompt template: %w", err)
	}

	return buf.String(), nil
}

// BugPromptData contains the data for rendering the bug prompt template
type BugPromptData struct {
	Title       string
	Description string
	Phase       string
	BugID       string
	BugDir      string
	PlanFile    string
	TasksFile   string
}

// LoadBugPrompt loads and renders the bug system prompt
// It checks for an external override at .sekkei/templates/bug-prompt.md first,
// then falls back to the embedded default prompt
func LoadBugPrompt(data BugPromptData) (string, error) {
	promptContent := defaultBugPrompt

	// Check for external override
	overridePath := filepath.Join(".sekkei", "templates", "bug-prompt.md")
	if content, err := os.ReadFile(overridePath); err == nil {
		promptContent = string(content)
	}

	// Parse and execute template
	tmpl, err := template.New("bug-prompt").Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("parse bug prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute bug prompt template: %w", err)
	}

	return buf.String(), nil
}
