package main

import (
	"os"

	"github.com/dphaener/shiki-cli/internal/cli"
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
		Long: `Generate shell completion scripts for shiki.

To load completions:

Bash:
  $ source <(shiki completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ shiki completion bash > /etc/bash_completion.d/shiki
  # macOS:
  $ shiki completion bash > $(brew --prefix)/etc/bash_completion.d/shiki

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Add to ~/.zshrc:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ shiki completion zsh > "${fpath[1]}/_shiki"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ shiki completion fish | source
  # To load completions for each session, execute once:
  $ shiki completion fish > ~/.config/fish/completions/shiki.fish

PowerShell:
  PS> shiki completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> shiki completion powershell > shiki.ps1
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
		Use:   "shiki",
		Short: "Spec-driven development with AI agent orchestration",
		Long: `shiki is a CLI tool for spec-driven development and AI agent orchestration.

It manages structured workflows (research → plan → implement → review → ship),
turn-based multi-agent collaboration, and real-time monitoring.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path (default: ~/.config/shiki-cli/config.json)")
	rootCmd.PersistentFlags().StringP("workspace", "w", "", "Workspace directory (default: ~/.local/share/shiki-cli/sessions)")

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
	rootCmd.AddCommand(cli.NewBugCommand())
	rootCmd.AddCommand(cli.NewVersionCommand(version, commit, buildDate))
	rootCmd.AddCommand(newCompletionCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}
