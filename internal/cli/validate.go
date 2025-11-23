package cli

import (
	"fmt"
	"os"

	"github.com/darinhaener/collab/internal/template"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates the validate command
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <task-file>",
		Short: "Validate a task template file",
		Long: `Validate a task template file without starting a session.

Checks for:
- Valid YAML frontmatter
- Required fields present
- Correct field types and ranges
- Valid workspace structure definitions`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskFile := args[0]

			// Check if file exists
			if _, err := os.Stat(taskFile); os.IsNotExist(err) {
				PrintError("Template file not found: %s", taskFile)
				return ExitWithCode(ExitValidation)
			}

			// Parse template
			tmpl, err := template.ParseTemplate(taskFile)
			if err != nil {
				PrintError("Failed to parse template:")
				fmt.Fprintf(os.Stderr, "  %v\n", err)
				return ExitWithCode(ExitValidation)
			}

			// Validate template
			if err := template.ValidateTemplate(tmpl); err != nil {
				PrintError("Template validation failed:")
				fmt.Fprintf(os.Stderr, "  %v\n", err)
				return ExitWithCode(ExitValidation)
			}

			// Success
			PrintSuccess("Template is valid: %s", taskFile)
			PrintInfo("Agent 1: %s (%s)", tmpl.Agent1Name, tmpl.Agent1Role)
			PrintInfo("Agent 2: %s (%s)", tmpl.Agent2Name, tmpl.Agent2Role)
			PrintInfo("Max turns: %d", tmpl.MaxTurns)
			if tmpl.WorkspaceStructure != nil && len(tmpl.WorkspaceStructure) > 0 {
				PrintInfo("Workspace structure: %d items defined", len(tmpl.WorkspaceStructure))
			}

			return nil
		},
	}

	return cmd
}
