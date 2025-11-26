package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
	"time"
)

//go:embed spec_template.md
var specTemplateContent string

//go:embed requirements_checklist.md
var checklistTemplateContent string

// SpecTemplateData holds data for rendering the spec template
type SpecTemplateData struct {
	FeatureName   string
	FeatureNumber string
	CreatedDate   string
}

// ChecklistTemplateData holds data for rendering the checklist template
type ChecklistTemplateData struct {
	FeatureName string
	CreatedDate string
}

// RenderSpecTemplate renders the spec template with provided data
func RenderSpecTemplate(featureName string, featureNumber int) (string, error) {
	tmpl, err := template.New("spec").Parse(specTemplateContent)
	if err != nil {
		return "", fmt.Errorf("parse spec template: %w", err)
	}

	data := SpecTemplateData{
		FeatureName:   featureName,
		FeatureNumber: fmt.Sprintf("%03d", featureNumber),
		CreatedDate:   time.Now().Format("2006-01-02"),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute spec template: %w", err)
	}

	return buf.String(), nil
}

// RenderChecklistTemplate renders the requirements checklist template
func RenderChecklistTemplate(featureName string) (string, error) {
	tmpl, err := template.New("checklist").Parse(checklistTemplateContent)
	if err != nil {
		return "", fmt.Errorf("parse checklist template: %w", err)
	}

	data := ChecklistTemplateData{
		FeatureName: featureName,
		CreatedDate: time.Now().Format("2006-01-02"),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute checklist template: %w", err)
	}

	return buf.String(), nil
}

// GetSpecTemplate returns the raw spec template content
func GetSpecTemplate() string {
	return specTemplateContent
}

// GetChecklistTemplate returns the raw checklist template content
func GetChecklistTemplate() string {
	return checklistTemplateContent
}
