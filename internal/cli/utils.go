package cli

import (
	"fmt"
	"os"

	"github.com/darinhaener/collab/pkg/types"
)

// Exit codes per CLI contract
const (
	ExitSuccess       = 0
	ExitIncomplete    = 1
	ExitError         = 2
	ExitValidation    = 3
	ExitUserInterrupt = 130
)

// exitWithCode creates an error that signals the desired exit code
type exitCodeError struct {
	code int
}

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

// ExitWithCode returns an error that will cause the program to exit with the given code
func ExitWithCode(code int) error {
	return &exitCodeError{code: code}
}

// handleExitCode checks if an error is an exitCodeError and exits with the appropriate code
func handleExitCode(err error) {
	if err == nil {
		return
	}

	if exitErr, ok := err.(*exitCodeError); ok {
		os.Exit(exitErr.code)
	}

	os.Exit(ExitError)
}

// SessionStatusExitCode returns the appropriate exit code for a session status
func SessionStatusExitCode(status types.SessionStatus) int {
	switch status {
	case types.SessionCompleted:
		return ExitSuccess
	case types.SessionIncomplete:
		return ExitIncomplete
	case types.SessionPaused:
		return ExitUserInterrupt
	default:
		return ExitError
	}
}

// PrintError prints an error message to stderr
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// PrintSuccess prints a success message with a checkmark
func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	fmt.Printf("⚠ "+format+"\n", args...)
}

// PrintInfo prints an informational message
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("  "+format+"\n", args...)
}
