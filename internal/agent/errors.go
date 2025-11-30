package agent

import "strings"

// IsBrokenPipeError checks if an error indicates a broken stdin pipe.
// This happens when the SDK client's underlying process has died or
// the pipe was closed (e.g., laptop sleep, idle timeout).
func IsBrokenPipeError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "failed to write to stdin") ||
		strings.Contains(errStr, "file already closed") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset")
}

// IsRecoverableError determines if client recreation would help.
// Returns true for broken pipe errors and timeout errors.
func IsRecoverableError(err error) bool {
	if err == nil {
		return false
	}
	if IsBrokenPipeError(err) {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "context deadline exceeded")
}
