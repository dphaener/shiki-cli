package template

import (
	"testing"

	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTemplateValid(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt 1",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt 2",
		MaxTurns:           10,
	}

	err := ValidateTemplate(tmpl)
	assert.NoError(t, err)
}

func TestValidateTemplateMissingRequired(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name: "Alice",
		// Missing other required fields
		MaxTurns: 10,
	}

	err := ValidateTemplate(tmpl)
	require.Error(t, err)

	verrs, ok := err.(ValidationErrors)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(verrs), 5) // Missing 5+ required fields
}

func TestValidateTemplateInvalidMaxTurns(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt",
		MaxTurns:           0, // Invalid
	}

	err := ValidateTemplate(tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_turns")
}

func TestValidateTemplateNegativeTimeout(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt",
		MaxTurns:           10,
		TurnTimeoutSeconds: -1, // Invalid
	}

	err := ValidateTemplate(tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "turn_timeout_seconds")
}

func TestValidateTemplateInvalidWorkspaceStructure(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt",
		MaxTurns:           10,
		WorkspaceStructure: []storage.WorkspaceStructure{
			{Path: "", Type: "file"}, // Empty path
			{Path: "test", Type: "invalid"}, // Invalid type
		},
	}

	err := ValidateTemplate(tmpl)
	require.Error(t, err)

	verrs, ok := err.(ValidationErrors)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(verrs), 2)
}

func TestValidateTemplateInvalidCostLimits(t *testing.T) {
	tests := []struct {
		name      string
		limits    *CostLimits
		errorMsg  string
	}{
		{
			name:     "negative per_agent",
			limits:   &CostLimits{PerAgent: -1, Total: 10},
			errorMsg: "per_agent",
		},
		{
			name:     "negative total",
			limits:   &CostLimits{PerAgent: 5, Total: -1},
			errorMsg: "total",
		},
		{
			name:     "per_agent exceeds total",
			limits:   &CostLimits{PerAgent: 15, Total: 10},
			errorMsg: "per_agent cannot exceed total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &TaskTemplate{
				Agent1Name:         "Alice",
				Agent1Role:         "Designer",
				Agent1SystemPrompt: "Prompt",
				Agent2Name:         "Bob",
				Agent2Role:         "Reviewer",
				Agent2SystemPrompt: "Prompt",
				MaxTurns:           10,
				CostLimits:         tt.limits,
			}

			err := ValidateTemplate(tmpl)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestValidateTemplateValidCostLimits(t *testing.T) {
	tmpl := &TaskTemplate{
		Agent1Name:         "Alice",
		Agent1Role:         "Designer",
		Agent1SystemPrompt: "Prompt",
		Agent2Name:         "Bob",
		Agent2Role:         "Reviewer",
		Agent2SystemPrompt: "Prompt",
		MaxTurns:           10,
		CostLimits: &CostLimits{
			PerAgent: 5.00,
			Total:    10.00,
		},
	}

	err := ValidateTemplate(tmpl)
	assert.NoError(t, err)
}
