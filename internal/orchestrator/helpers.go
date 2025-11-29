package orchestrator

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// MessageUpdate represents a streaming update from the agent
type MessageUpdate struct {
	Type    string                 // "text", "tool_use", "tool_error", "message_done", "complete"
	Content string                 // Text content or tool name
	Args    map[string]interface{} // Tool arguments (for tool_use type)
}

// maxRetries is the maximum number of times to retry on timeout
const maxRetries = 3

// retryDelay is the delay between retry attempts
const retryDelay = 2 * time.Second

// interruptTimeout is the maximum time to wait for an interrupt call to complete.
// If the SDK's Interrupt method hangs, we don't want to block forever.
const interruptTimeout = 10 * time.Second

// sdkClient interface for interrupt capability (allows testing)
type sdkClient interface {
	Interrupt(ctx context.Context) error
}

// interruptClient attempts to interrupt the SDK client with a timeout.
// This prevents hanging if the SDK's Interrupt method blocks.
// Returns true if interrupt succeeded, false if it timed out or failed.
func interruptClient(client sdkClient, logger *debugLogger) bool {
	// Create a timeout context for the interrupt
	ctx, cancel := context.WithTimeout(context.Background(), interruptTimeout)
	defer cancel()

	// Channel to signal interrupt completion
	done := make(chan error, 1)
	go func() {
		done <- client.Interrupt(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			if logger != nil {
				logger.log("Interrupt failed (may be expected): %v", err)
			}
		}
		return err == nil
	case <-ctx.Done():
		if logger != nil {
			logger.log("Interrupt timed out after %v - forcing continuation", interruptTimeout)
		}
		return false
	}
}

// Pre-compiled regex patterns for error message cleaning (compiled once at package init)
var (
	// Stack traces (lines starting with 'at ' or containing file paths)
	stackTraceLineRegex  = regexp.MustCompile(`(?m)^\s*at\s+.*$`)
	stackTraceEntryRegex = regexp.MustCompile(`(?m)^\s*\w+\.\w+\([^)]*\):\d+.*$`)

	// File paths - more specific patterns
	absoluteFilePathRegex = regexp.MustCompile(`/[\w\-./]+\.\w+`)
	stackTraceAtRegex     = regexp.MustCompile(`\s+at\s+[\w./\\]+:\d+`)

	// Error codes - more specific to avoid false positives
	errorCodeLabelRegex = regexp.MustCompile(`(?i)\berror code[:\s]+[A-Za-z0-9_]+`)
	errorCodeConstRegex = regexp.MustCompile(`\b[A-Z]{3,}_[A-Z0-9_]+\b`)
	hexValueRegex       = regexp.MustCompile(`\b0x[0-9A-Fa-f]+\b`)

	// Debug information and verbose details
	stackTraceKeywordRegex = regexp.MustCompile(`\bstack trace:.*`)
	traceKeywordRegex      = regexp.MustCompile(`\btrace:.*`)
	debugKeywordRegex      = regexp.MustCompile(`\bdebug:\s*[^\n]*`)

	// Technical JSON/XML fragments
	jsonFragmentRegex = regexp.MustCompile(`\{[^{}]*"[^"]*"[^{}]*\}`)
	xmlTagRegex       = regexp.MustCompile(`<[^>]*>`)

	// Clean up multiple whitespace but preserve structure
	multipleBlankLinesRegex = regexp.MustCompile(`\n\s*\n\s*\n+`)
	multipleSpacesRegex     = regexp.MustCompile(`\s+`)
)

// errorCleaningPattern pairs a pre-compiled regex with its replacement string
type errorCleaningPattern struct {
	regex       *regexp.Regexp
	replacement string
}

// errorCleaningPatterns is the ordered list of patterns to apply for cleaning error messages
var errorCleaningPatterns = []errorCleaningPattern{
	{stackTraceLineRegex, ""},
	{stackTraceEntryRegex, ""},
	{absoluteFilePathRegex, ""},
	{stackTraceAtRegex, ""},
	{errorCodeLabelRegex, ""},
	{errorCodeConstRegex, ""},
	{hexValueRegex, ""},
	{stackTraceKeywordRegex, ""},
	{traceKeywordRegex, ""},
	{debugKeywordRegex, ""},
	{jsonFragmentRegex, ""},
	{xmlTagRegex, ""},
	{multipleBlankLinesRegex, "\n"},
	{multipleSpacesRegex, " "},
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// cleanErrorMessage sanitizes error messages by removing technical details
// and providing user-friendly error descriptions
func cleanErrorMessage(errorMsg string) string {
	if errorMsg == "" {
		return "An unknown error occurred"
	}

	cleaned := errorMsg

	// Apply all pre-compiled cleaning patterns
	for _, p := range errorCleaningPatterns {
		cleaned = p.regex.ReplaceAllString(cleaned, p.replacement)
	}

	// Clean up the result
	cleaned = strings.TrimSpace(cleaned)

	// Remove leading/trailing punctuation artifacts
	cleaned = strings.Trim(cleaned, ":.,; ")

	// Handle empty or too short result after cleaning
	if cleaned == "" || len(cleaned) < 3 {
		return "An error occurred. Please check your input and try again."
	}

	// Ensure the message starts with a capital letter and ends properly
	cleaned = capitalizeFirst(cleaned)
	if !strings.HasSuffix(cleaned, ".") && !strings.HasSuffix(cleaned, "!") && !strings.HasSuffix(cleaned, "?") {
		cleaned += "."
	}

	return cleaned
}

// formatErrorContext creates a user-friendly context description for tool errors
func formatErrorContext(toolName, toolArgs string) string {
	if toolName == "" {
		return ""
	}

	// Map technical tool names to user-friendly descriptions
	friendlyNames := map[string]string{
		"bash":       "running command",
		"edit":       "editing file",
		"write":      "writing file",
		"read":       "reading file",
		"grep":       "searching content",
		"glob":       "finding files",
		"web_fetch":  "fetching web content",
		"web_search": "searching web",
	}

	friendlyName := friendlyNames[strings.ToLower(toolName)]
	if friendlyName == "" {
		friendlyName = strings.ToLower(toolName)
	}

	return fmt.Sprintf("Error while %s", friendlyName)
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// GetAPIKey gets the API key from environment or returns empty string
func GetAPIKey() string {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		// Try to use Claude Code's saved credentials
		// The SDK will handle this automatically
		return ""
	}
	return apiKey
}
