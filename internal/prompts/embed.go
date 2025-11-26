package prompts

import _ "embed"

//go:embed specify_prompt.md
var defaultSpecifyPrompt string

//go:embed plan_prompt.md
var defaultPlanPrompt string
