package template

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidationErrors is a collection of validation failures
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var msgs []string
	for _, err := range ve {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "\n")
}

// ValidateTemplate checks all requirements
func ValidateTemplate(tmpl *TaskTemplate) error {
	var errors ValidationErrors

	// Required fields (FR-057)
	if tmpl.Agent1Name == "" {
		errors = append(errors, ValidationError{"agent1_name", "required field missing"})
	}
	if tmpl.Agent1Role == "" {
		errors = append(errors, ValidationError{"agent1_role", "required field missing"})
	}
	if tmpl.Agent1SystemPrompt == "" {
		errors = append(errors, ValidationError{"agent1_system_prompt", "required field missing"})
	}

	if tmpl.Agent2Name == "" {
		errors = append(errors, ValidationError{"agent2_name", "required field missing"})
	}
	if tmpl.Agent2Role == "" {
		errors = append(errors, ValidationError{"agent2_role", "required field missing"})
	}
	if tmpl.Agent2SystemPrompt == "" {
		errors = append(errors, ValidationError{"agent2_system_prompt", "required field missing"})
	}

	if tmpl.MaxTurns <= 0 {
		errors = append(errors, ValidationError{"max_turns", "must be greater than 0"})
	}

	// Validate optional fields if present
	if tmpl.TurnTimeoutSeconds < 0 {
		errors = append(errors, ValidationError{"turn_timeout_seconds", "cannot be negative"})
	}

	if tmpl.TurnDelayMS < 0 {
		errors = append(errors, ValidationError{"turn_delay_ms", "cannot be negative"})
	}

	// Validate workspace structure if provided
	for i, item := range tmpl.WorkspaceStructure {
		if item.Path == "" {
			errors = append(errors, ValidationError{
				fmt.Sprintf("workspace_structure[%d].path", i),
				"path cannot be empty",
			})
		}
		if item.Type != "file" && item.Type != "directory" {
			errors = append(errors, ValidationError{
				fmt.Sprintf("workspace_structure[%d].type", i),
				fmt.Sprintf("must be 'file' or 'directory', got '%s'", item.Type),
			})
		}
	}

	// Validate cost limits if provided
	if tmpl.CostLimits != nil {
		if tmpl.CostLimits.PerAgent < 0 {
			errors = append(errors, ValidationError{"cost_limits.per_agent", "cannot be negative"})
		}
		if tmpl.CostLimits.Total < 0 {
			errors = append(errors, ValidationError{"cost_limits.total", "cannot be negative"})
		}
		if tmpl.CostLimits.Total > 0 && tmpl.CostLimits.PerAgent > tmpl.CostLimits.Total {
			errors = append(errors, ValidationError{"cost_limits", "per_agent cannot exceed total"})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}
