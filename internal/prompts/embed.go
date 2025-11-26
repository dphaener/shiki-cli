package prompts

import _ "embed"

//go:embed specify_prompt.md
var defaultSpecifyPrompt string

//go:embed plan_prompt.md
var defaultPlanPrompt string

//go:embed tasks_prompt.md
var defaultTasksPrompt string

//go:embed implement_prompt.md
var defaultImplementPrompt string
