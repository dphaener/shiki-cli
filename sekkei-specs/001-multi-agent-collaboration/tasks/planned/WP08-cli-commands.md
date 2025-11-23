---
work_package_id: WP08
title: "CLI Commands"
priority: P2
status: planned
subtasks:
  - T042
  - T043
  - T044
  - T045
  - T046
  - T047
  - T048
  - T049
  - T050
  - T051
dependencies:
  - WP01
  - WP04
  - WP07
  - WP09
lane: planned
history:
  - timestamp: "2025-11-23"
    action: created
    status: planned
---

# Work Package WP08: CLI Commands

**Objective**: Implement all Cobra commands with flag parsing, output formatting (table/json/csv), and proper exit codes per CLI interface contract.

**Priority**: P2 (User interface commands)

**Estimated Effort**: 8-10 hours

## Context

The CLI is the user-facing interface providing 9 commands (run, resume, list, show, watch, validate, init, clean, version) with consistent flag handling and output formatting.

**Exit Codes** (contracts/cli-interface.md:531-539):
- 0: Success
- 1: Session incomplete (max turns)
- 2: Error (crash, API failure)
- 3: Validation error
- 130: User interrupted (Ctrl+C)

## Detailed Guidance

### T042: Implement root command and main.go

Create `cmd/collab/main.go`:
```go
package main

import (
    "os"
    "github.com/spf13/cobra"
    "github.com/yourusername/collab/internal/cli"
)

var (
    version   = "dev"
    commit    = "unknown"
    buildDate = "unknown"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "collab",
        Short: "Multi-agent AI collaboration orchestrator",
        Long:  "CLI tool for orchestrating two-agent AI collaborations via file-based communication",
    }
    
    // Global flags
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
    rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
    rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path")
    rootCmd.PersistentFlags().StringP("workspace", "w", "", "Workspace directory")
    
    // Add commands
    rootCmd.AddCommand(cli.NewRunCommand())
    rootCmd.AddCommand(cli.NewResumeCommand())
    rootCmd.AddCommand(cli.NewListCommand())
    rootCmd.AddCommand(cli.NewShowCommand())
    rootCmd.AddCommand(cli.NewWatchCommand())
    rootCmd.AddCommand(cli.NewValidateCommand())
    rootCmd.AddCommand(cli.NewInitCommand())
    rootCmd.AddCommand(cli.NewCleanCommand())
    rootCmd.AddCommand(cli.NewVersionCommand(version, commit, buildDate))
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(2)
    }
}
```

### T043-T050: Individual commands

Create `internal/cli/run.go`:
```go
package cli

import (
    "context"
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/yourusername/collab/internal/template"
    "github.com/yourusername/collab/internal/orchestrator"
)

func NewRunCommand() *cobra.Command {
    var watchMode bool
    
    cmd := &cobra.Command{
        Use:   "run <task-file>",
        Short: "Start a new collaboration session",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            taskFile := args[0]
            
            // Parse and validate template
            tmpl, err := template.ParseTemplate(taskFile)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
                return exitWithCode(3)
            }
            
            if err := template.ValidateTemplate(tmpl); err != nil {
                fmt.Fprintf(os.Stderr, "Template validation failed:\n%v\n", err)
                return exitWithCode(3)
            }
            
            // Create session and workspace
            session := createSession(tmpl)
            
            // Start orchestrator
            orch := orchestrator.New(session)
            
            if watchMode {
                // Launch TUI
                return runWithTUI(orch)
            } else {
                // Run headless
                return runHeadless(orch)
            }
        },
    }
    
    cmd.Flags().BoolVar(&watchMode, "watch", false, "Launch interactive TUI")
    
    return cmd
}

func runHeadless(orch *orchestrator.Orchestrator) error {
    ctx := context.Background()
    
    if err := orch.Run(ctx); err != nil {
        return exitWithCode(2)
    }
    
    // Check final status
    switch orch.Session().Status {
    case types.SessionCompleted:
        fmt.Printf("✓ Session completed\n")
        fmt.Printf("✓ Deliverable: %s\n", orch.Session().DeliverablePath)
        return exitWithCode(0)
    case types.SessionIncomplete:
        fmt.Printf("✗ Session incomplete (max turns reached)\n")
        return exitWithCode(1)
    case types.SessionPaused:
        fmt.Printf("⏸ Session paused\n")
        return exitWithCode(130)
    default:
        return exitWithCode(2)
    }
}
```

Similarly implement:
- `cli/resume.go` (T044)
- `cli/list.go` (T045) - with table/json/csv formatting
- `cli/show.go` (T046) - with --messages, --deliverable flags
- `cli/validate.go` (T047)
- `cli/init.go` (T048) - interactive wizard
- `cli/clean.go` (T049) - with confirmation prompt
- `cli/version.go` (T050)
- `cli/watch.go` (T051) - depends on WP09 TUI

## Test Strategy

Integration tests for each command:
```go
func TestRunCommand(t *testing.T) {
    // Create temp template
    tmpFile := createTempTemplate(t)
    
    // Execute command
    cmd := NewRunCommand()
    cmd.SetArgs([]string{tmpFile})
    
    err := cmd.Execute()
    require.NoError(t, err)
}
```

## Definition of Done

- [ ] All 10 subtasks completed
- [ ] All commands support global flags
- [ ] Exit codes match contract specifications
- [ ] Output formatting works (table/json/csv)
- [ ] Help text auto-generated by Cobra
- [ ] Integration tests for each command

## References

- [contracts/cli-interface.md](../contracts/cli-interface.md): Complete CLI specification
- [spec.md](../spec.md): FR-001 to FR-010
