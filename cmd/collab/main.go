package main

import (
	"fmt"
	"os"

	"github.com/darinhaener/collab/internal/config"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	fmt.Printf("collab version %s\n", version)
	fmt.Printf("commit: %s, built: %s\n", commit, buildDate)

	cfg, err := config.Load("", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("workspace: %s\n", cfg.WorkspaceDir)
	fmt.Printf("model: %s\n", cfg.DefaultModel)
}
