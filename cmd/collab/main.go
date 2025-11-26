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

func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for collab.

To load completions:

Bash:
  $ source <(collab completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ collab completion bash > /etc/bash_completion.d/collab
  # macOS:
  $ collab completion bash > $(brew --prefix)/etc/bash_completion.d/collab

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Add to ~/.zshrc:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ collab completion zsh > "${fpath[1]}/_collab"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ collab completion fish | source
  # To load completions for each session, execute once:
  $ collab completion fish > ~/.config/fish/completions/collab.fish

PowerShell:
  PS> collab completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> collab completion powershell > collab.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}

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
	rootCmd.AddCommand(cli.NewSpecifyCommand())
	rootCmd.AddCommand(cli.NewPlanCommand())
	rootCmd.AddCommand(cli.NewFeatureCommand())
	rootCmd.AddCommand(cli.NewVersionCommand(version, commit, buildDate))
	rootCmd.AddCommand(newCompletionCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}
