package template

import "github.com/darinhaener/collab/internal/storage"

// TaskTemplate represents a task template with YAML frontmatter
type TaskTemplate struct {
	// Agent 1 configuration
	Agent1Name         string `yaml:"agent1_name"`
	Agent1Role         string `yaml:"agent1_role"`
	Agent1SystemPrompt string `yaml:"agent1_system_prompt"`
	Agent1Model        string `yaml:"agent1_model"`

	// Agent 2 configuration
	Agent2Name         string `yaml:"agent2_name"`
	Agent2Role         string `yaml:"agent2_role"`
	Agent2SystemPrompt string `yaml:"agent2_system_prompt"`
	Agent2Model        string `yaml:"agent2_model"`

	// Collaboration settings
	MaxTurns           int `yaml:"max_turns"`
	TurnTimeoutSeconds int `yaml:"turn_timeout_seconds"`
	TurnDelayMS        int `yaml:"turn_delay_ms"`

	// Optional workspace structure
	WorkspaceStructure []storage.WorkspaceStructure `yaml:"workspace_structure,omitempty"`

	// Optional completion criteria
	CompletionCriteria string `yaml:"completion_criteria,omitempty"`

	// Optional cost limits
	CostLimits *CostLimits `yaml:"cost_limits,omitempty"`

	// Task body (markdown after frontmatter)
	TaskBody string `yaml:"-"`
}

// CostLimits defines cost constraints for the collaboration
type CostLimits struct {
	PerAgent float64 `yaml:"per_agent"`
	Total    float64 `yaml:"total"`
}

// Defaults returns a template with sensible defaults
func Defaults() *TaskTemplate {
	return &TaskTemplate{
		Agent1Model:        "claude-sonnet-4",
		Agent2Model:        "claude-sonnet-4",
		TurnTimeoutSeconds: 300,
		TurnDelayMS:        0,
	}
}
