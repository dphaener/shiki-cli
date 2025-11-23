package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var (
		interactive bool
		outputPath  string
	)

	cmd := &cobra.Command{
		Use:   "init [template-name]",
		Short: "Initialize a new task template",
		Long: `Create a new task template file from scratch or from an example.

Without arguments, creates a minimal template. With a template name, copies
an example template. Use --interactive for a guided wizard.

Available example templates:
  - simple-agreement: Two agents reach consensus on a simple task
  - code-review: Architect and reviewer collaborate on code
  - architecture-design: Design system architecture collaboratively`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var templateName string
			if len(args) > 0 {
				templateName = args[0]
			}

			if interactive {
				return runInteractiveWizard(outputPath)
			}

			if templateName != "" {
				return copyExampleTemplate(templateName, outputPath)
			}

			return createMinimalTemplate(outputPath)
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run interactive template wizard")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: task.md)")

	return cmd
}

// runInteractiveWizard runs an interactive template creation wizard
func runInteractiveWizard(outputPath string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Task Template Wizard ===")
	fmt.Println()

	// Get agent 1 details
	fmt.Print("Agent 1 name: ")
	agent1Name, _ := reader.ReadString('\n')
	agent1Name = strings.TrimSpace(agent1Name)

	fmt.Print("Agent 1 role: ")
	agent1Role, _ := reader.ReadString('\n')
	agent1Role = strings.TrimSpace(agent1Role)

	fmt.Print("Agent 1 system prompt: ")
	agent1Prompt, _ := reader.ReadString('\n')
	agent1Prompt = strings.TrimSpace(agent1Prompt)

	// Get agent 2 details
	fmt.Print("Agent 2 name: ")
	agent2Name, _ := reader.ReadString('\n')
	agent2Name = strings.TrimSpace(agent2Name)

	fmt.Print("Agent 2 role: ")
	agent2Role, _ := reader.ReadString('\n')
	agent2Role = strings.TrimSpace(agent2Role)

	fmt.Print("Agent 2 system prompt: ")
	agent2Prompt, _ := reader.ReadString('\n')
	agent2Prompt = strings.TrimSpace(agent2Prompt)

	// Get max turns
	fmt.Print("Maximum turns (default: 10): ")
	maxTurnsStr, _ := reader.ReadString('\n')
	maxTurnsStr = strings.TrimSpace(maxTurnsStr)
	if maxTurnsStr == "" {
		maxTurnsStr = "10"
	}

	// Get task description
	fmt.Print("Task description: ")
	taskDesc, _ := reader.ReadString('\n')
	taskDesc = strings.TrimSpace(taskDesc)

	// Create template content
	template := fmt.Sprintf(`---
agent1_name: %s
agent1_role: %s
agent1_system_prompt: |
  %s
agent1_model: claude-sonnet-4

agent2_name: %s
agent2_role: %s
agent2_system_prompt: |
  %s
agent2_model: claude-sonnet-4

max_turns: %s

workspace_structure:
  shared_context.md: ""
  messages.md: ""
---

# Task: %s

%s
`,
		agent1Name, agent1Role, agent1Prompt,
		agent2Name, agent2Role, agent2Prompt,
		maxTurnsStr,
		taskDesc, taskDesc)

	// Determine output path
	if outputPath == "" {
		outputPath = "task.md"
	}

	// Write template
	if err := os.WriteFile(outputPath, []byte(template), 0644); err != nil {
		PrintError("Failed to write template: %v", err)
		return ExitWithCode(ExitError)
	}

	PrintSuccess("Template created: %s", outputPath)
	PrintInfo("Validate with: collab validate %s", outputPath)
	PrintInfo("Run with: collab run %s", outputPath)

	return nil
}

// copyExampleTemplate copies an example template
func copyExampleTemplate(templateName, outputPath string) error {
	// In a real implementation, this would copy from embedded templates
	// For now, just create a placeholder
	PrintError("Example template '%s' not found", templateName)
	PrintInfo("Available templates: simple-agreement, code-review, architecture-design")
	return ExitWithCode(ExitError)
}

// createMinimalTemplate creates a minimal template
func createMinimalTemplate(outputPath string) error {
	template := `---
agent1_name: Agent One
agent1_role: Collaborator
agent1_system_prompt: |
  You are Agent One. Work with Agent Two to complete the task.
agent1_model: claude-sonnet-4

agent2_name: Agent Two
agent2_role: Partner
agent2_system_prompt: |
  You are Agent Two. Work with Agent One to complete the task.
agent2_model: claude-sonnet-4

max_turns: 10

workspace_structure:
  shared_context.md: ""
  messages.md: ""
---

# Task: Collaboration Task

Complete this task by collaborating with your partner agent.
`

	if outputPath == "" {
		outputPath = "task.md"
	}

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		PrintError("File already exists: %s", outputPath)
		PrintInfo("Use --output to specify a different path")
		return ExitWithCode(ExitError)
	}

	if err := os.WriteFile(outputPath, []byte(template), 0644); err != nil {
		PrintError("Failed to write template: %v", err)
		return ExitWithCode(ExitError)
	}

	PrintSuccess("Minimal template created: %s", outputPath)
	PrintInfo("Edit the template to customize agents and task")
	PrintInfo("Validate with: collab validate %s", outputPath)

	return nil
}
