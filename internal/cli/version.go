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
			fmt.Printf("collab version %s\n", version)
			fmt.Printf("  commit:     %s\n", commit)
			fmt.Printf("  built:      %s\n", buildDate)
			fmt.Printf("  go version: %s\n", runtime.Version())
			fmt.Printf("  platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}

	return cmd
}
