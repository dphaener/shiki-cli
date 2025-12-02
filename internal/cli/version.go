package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command
func NewVersionCommand(version, commit, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  "Display version, build commit, build date, and Go version information",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "shiki version %s\n", version)
			fmt.Fprintf(out, "  commit:     %s\n", commit)
			fmt.Fprintf(out, "  built:      %s\n", buildDate)
			fmt.Fprintf(out, "  go version: %s\n", runtime.Version())
			fmt.Fprintf(out, "  platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}

	return cmd
}
