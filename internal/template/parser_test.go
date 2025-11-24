package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplate(t *testing.T) {
	template := `---
agent1_name: "Alice"
agent1_role: "Designer"
agent1_system_prompt: "You are a designer"
agent2_name: "Bob"
agent2_role: "Reviewer"
agent2_system_prompt: "You are a reviewer"
max_turns: 5
---

# Task

Design a system.
`

	tmpl, err := ParseTemplateBytes([]byte(template))
	require.NoError(t, err)

	assert.Equal(t, "Alice", tmpl.Agent1Name)
	assert.Equal(t, "Bob", tmpl.Agent2Name)
	assert.Equal(t, 5, tmpl.MaxTurns)
	assert.Equal(t, "# Task\n\nDesign a system.", tmpl.TaskBody)
	assert.Equal(t, "claude-sonnet-4-20250514", tmpl.Agent1Model) // Default
}

func TestParseTemplateMalformedYAML(t *testing.T) {
	template := `---
agent1_name: "Alice
invalid yaml here
---

# Task
`

	_, err := ParseTemplateBytes([]byte(template))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML frontmatter error")
}

func TestParseTemplateMissingFrontmatter(t *testing.T) {
	template := `# Task without frontmatter`

	_, err := ParseTemplateBytes([]byte(template))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing YAML frontmatter")
}

func TestParseTemplateWithWorkspaceStructure(t *testing.T) {
	template := `---
agent1_name: "Alice"
agent1_role: "Designer"
agent1_system_prompt: "You are a designer"
agent2_name: "Bob"
agent2_role: "Reviewer"
agent2_system_prompt: "You are a reviewer"
max_turns: 5
workspace_structure:
  - path: "src/"
    type: "directory"
  - path: "README.md"
    type: "file"
    content: "# Project\n"
---

# Task

Design a system.
`

	tmpl, err := ParseTemplateBytes([]byte(template))
	require.NoError(t, err)

	assert.Len(t, tmpl.WorkspaceStructure, 2)
	assert.Equal(t, "src/", tmpl.WorkspaceStructure[0].Path)
	assert.Equal(t, "directory", tmpl.WorkspaceStructure[0].Type)
	assert.Equal(t, "README.md", tmpl.WorkspaceStructure[1].Path)
	assert.Equal(t, "file", tmpl.WorkspaceStructure[1].Type)
	assert.Equal(t, "# Project\n", tmpl.WorkspaceStructure[1].Content)
}

func TestParseTemplateWithCostLimits(t *testing.T) {
	template := `---
agent1_name: "Alice"
agent1_role: "Designer"
agent1_system_prompt: "You are a designer"
agent2_name: "Bob"
agent2_role: "Reviewer"
agent2_system_prompt: "You are a reviewer"
max_turns: 5
cost_limits:
  per_agent: 2.50
  total: 5.00
---

# Task

Design a system.
`

	tmpl, err := ParseTemplateBytes([]byte(template))
	require.NoError(t, err)

	require.NotNil(t, tmpl.CostLimits)
	assert.Equal(t, 2.50, tmpl.CostLimits.PerAgent)
	assert.Equal(t, 5.00, tmpl.CostLimits.Total)
}
