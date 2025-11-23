package main

import (
	"os"

	"github.com/darinhaener/collab/internal/cli"
	"github.com/spf13/cobra"
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
		Long: `collab is a CLI tool for orchestrating two-agent AI collaborations via file-based communication.

It manages turn-based execution, real-time monitoring, and automatic completion detection
when both agents reach consensus on a deliverable.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path (default: ~/.config/collab-cli/config.json)")
	rootCmd.PersistentFlags().StringP("workspace", "w", "", "Workspace directory (default: ~/.local/share/collab-cli/sessions)")

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
