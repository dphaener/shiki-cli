package orchestrator

import (
	"strings"
	"testing"
)

func TestCleanErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty error message",
			input:    "",
			expected: "An unknown error occurred",
		},
		{
			name:     "Simple error message",
			input:    "file not found",
			expected: "File not found.",
		},
		{
			name:     "Error with stack trace",
			input:    "Permission denied\nat main.go:42\nat function.call(args)",
			expected: "Permission denied.",
		},
		{
			name:     "Error with file paths",
			input:    "Cannot read /usr/local/bin/file.txt: permission denied",
			expected: "Cannot read : permission denied.",
		},
		{
			name:     "Error with error codes",
			input:    "HTTP_ERROR_404 not found error code ENOENT",
			expected: "Not found.",
		},
		{
			name:     "Error with debug information",
			input:    "Connection failed debug: timeout after 30s trace: network unreachable",
			expected: "Connection failed.",
		},
		{
			name:     "Error with JSON fragments",
			input:    `API error {"status":"failed","code":500} occurred`,
			expected: "API error occurred.",
		},
		{
			name:     "Complex error with multiple patterns",
			input:    "Tool execution failed\nat /usr/local/bin/tool.py:123\nError code: ERR_TIMEOUT\ndebug: connection lost\n{\"error\":\"network\"}\nstack trace: ...",
			expected: "Tool execution failed.",
		},
		{
			name:     "Error that becomes empty after cleaning",
			input:    "at main.go:42\nerror code: ABC_123\ndebug: info",
			expected: "An error occurred. Please check your input and try again.",
		},
		{
			name:     "Error with proper punctuation already",
			input:    "Invalid input provided!",
			expected: "Invalid input provided!",
		},
		{
			name:     "Error with question mark",
			input:    "Are you sure about this configuration?",
			expected: "Are you sure about this configuration?",
		},
		{
			name:     "Multiline error with whitespace",
			input:    "Connection failed\n\n\nRetrying...\n   \nTimeout",
			expected: "Connection failed Retrying... Timeout.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("cleanErrorMessage(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatErrorContext(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		toolArgs string
		expected string
	}{
		{
			name:     "Empty tool name",
			toolName: "",
			toolArgs: "any args",
			expected: "",
		},
		{
			name:     "Known tool name - bash",
			toolName: "bash",
			toolArgs: `{"command": "ls -la"}`,
			expected: "Error while running command",
		},
		{
			name:     "Known tool name - edit",
			toolName: "edit",
			toolArgs: `{"file": "test.go"}`,
			expected: "Error while editing file",
		},
		{
			name:     "Known tool name - case insensitive",
			toolName: "BASH",
			toolArgs: "args",
			expected: "Error while running command",
		},
		{
			name:     "Unknown tool name",
			toolName: "custom_tool",
			toolArgs: "args",
			expected: "Error while custom_tool",
		},
		{
			name:     "Web fetch tool",
			toolName: "web_fetch",
			toolArgs: `{"url": "http://example.com"}`,
			expected: "Error while fetching web content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorContext(tt.toolName, tt.toolArgs)
			if result != tt.expected {
				t.Errorf("formatErrorContext(%q, %q) = %q, expected %q", tt.toolName, tt.toolArgs, result, tt.expected)
			}
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Single character lowercase",
			input:    "a",
			expected: "A",
		},
		{
			name:     "Single character uppercase",
			input:    "A",
			expected: "A",
		},
		{
			name:     "Normal sentence",
			input:    "hello world",
			expected: "Hello world",
		},
		{
			name:     "Already capitalized",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Number first",
			input:    "123 error",
			expected: "123 error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalizeFirst(tt.input)
			if result != tt.expected {
				t.Errorf("capitalizeFirst(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCleanErrorMessagePreservesEssentialInfo ensures that essential error information
// is not removed during the cleaning process
func TestCleanErrorMessagePreservesEssentialInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // Strings that should be preserved
		notContains []string // Strings that should be removed
	}{
		{
			name:     "Permission error preserves main message",
			input:    "Permission denied accessing file /path/to/file.txt at main.go:42",
			contains: []string{"Permission denied", "accessing file"},
			notContains: []string{"/path/to/file.txt", "main.go:42"},
		},
		{
			name:     "Network error preserves meaningful content",
			input:    "Connection timeout after 30 seconds\ndebug: retry count 3\nat network.go:123",
			contains: []string{"Connection timeout", "30 seconds"},
			notContains: []string{"debug:", "network.go:123"},
		},
		{
			name:     "API error preserves user-actionable information",
			input:    "Invalid API key provided error code AUTH_001 {\"status\":\"failed\"}",
			contains: []string{"Invalid API key provided"},
			notContains: []string{"AUTH_001", `{"status":"failed"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanErrorMessage(tt.input)

			// Check that essential information is preserved
			for _, contain := range tt.contains {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(contain)) {
					t.Errorf("cleanErrorMessage(%q) = %q, should contain %q", tt.input, result, contain)
				}
			}

			// Check that technical details are removed
			for _, notContain := range tt.notContains {
				if strings.Contains(result, notContain) {
					t.Errorf("cleanErrorMessage(%q) = %q, should not contain %q", tt.input, result, notContain)
				}
			}
		})
	}
}

// TestCleanErrorMessageLongMessages tests behavior with very long error messages
func TestCleanErrorMessageLongMessages(t *testing.T) {
	// Create a long error message with 500+ characters
	longError := strings.Repeat("This is a very long error message with technical details. ", 10) +
		"at /very/long/path/to/file.go:999\n" +
		"Error code: SUPER_LONG_ERROR_CODE_123\n" +
		"debug: extensive debugging information here\n" +
		`{"error": "detailed json error", "code": 500, "trace": "long trace info"}`

	result := cleanErrorMessage(longError)

	// Should be significantly shorter
	if len(result) >= len(longError) {
		t.Errorf("cleanErrorMessage should reduce length of long messages, got %d chars from %d chars", len(result), len(longError))
	}

	// Should not contain technical details
	technicalPatterns := []string{"/very/long/path", "SUPER_LONG_ERROR_CODE", "debug:", `{"error"`}
	for _, pattern := range technicalPatterns {
		if strings.Contains(result, pattern) {
			t.Errorf("cleanErrorMessage should remove technical details like %q, but result contains it: %q", pattern, result)
		}
	}

	// Should still be readable and informative
	if len(result) < 20 {
		t.Errorf("cleanErrorMessage result too short, may have removed essential information: %q", result)
	}
}

// TestCleanErrorMessageSpecialCharacters tests handling of Unicode and special characters
func TestCleanErrorMessageSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unicode characters preserved",
			input:    "Error: файл не найден (file not found)",
			expected: "Error: файл не найден (file not found).",
		},
		{
			name:     "Emoji and symbols preserved",
			input:    "Error ❌: connection failed 🔌",
			expected: "Error ❌: connection failed 🔌.",
		},
		{
			name:     "Mixed special characters",
			input:    "Error: couldn't access file @#$%^&*()",
			expected: "Error: couldn't access file @#$%^&*().",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("cleanErrorMessage(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}