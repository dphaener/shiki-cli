package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dphaener/shiki-cli/internal/config"
)

// debugLogger handles debug logging to a file
type debugLogger struct {
	file *os.File
}

// newDebugLogger creates a new debug logger with XDG-compliant path
func newDebugLogger() *debugLogger {
	// Attempt migration first
	if err := config.MigrateDebugLog(); err != nil {
		// Migration failed, but continue with current behavior
		fmt.Fprintf(os.Stderr, "Warning: debug log migration failed: %v\n", err)
	}

	// Get XDG-compliant log path
	logPath, err := config.GetDebugLogPath()
	if err != nil {
		// Fall back to current directory
		logPath = filepath.Join(".", "collab-debug.log")
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return &debugLogger{file: nil}
	}
	return &debugLogger{file: f}
}

// log writes a message to the debug log
func (d *debugLogger) log(format string, args ...interface{}) {
	if d.file == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(d.file, "[%s] %s\n", timestamp, msg)
	d.file.Sync()
}

// close closes the debug log file
func (d *debugLogger) close() {
	if d.file != nil {
		d.file.Close()
	}
}