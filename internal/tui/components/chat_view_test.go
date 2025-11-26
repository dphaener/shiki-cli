package components

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/darinhaener/collab/pkg/types"
)

// TestChatViewErrorRendering tests that error messages are properly rendered with text wrapping
func TestChatViewErrorRendering(t *testing.T) {
	// Create a new chat view with a specific width
	chatView := NewChatView(100, 20) // 100 width, 20 height

	// Test error message that should wrap
	longError := "This is a very long error message that should wrap properly across multiple lines when displayed in the terminal to ensure good user experience"

	// Create an error message
	errorMsg := types.ChatMessage{
		Role:      "error",
		Content:   longError,
		Timestamp: time.Now(),
	}

	// Render the error message
	rendered := chatView.renderMessage(errorMsg)

	// Check that the message was rendered (not empty)
	if rendered == "" {
		t.Error("Error message should not render as empty")
	}

	// Check that it contains the error icon
	if !strings.Contains(rendered, "⚠") {
		t.Error("Error message should contain warning icon")
	}

	// Check that it contains "Error" in the header
	if !strings.Contains(rendered, "Error") {
		t.Error("Error message should contain 'Error' text")
	}

	// Check that the content is included
	if !strings.Contains(rendered, "very long error message") {
		t.Error("Error message should contain the original content")
	}

	// The rendered message should not exceed the terminal width when each line is considered
	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		// Note: We can't easily test exact width without rendering styles,
		// but we can check that extremely long lines are not present
		if len(line) > 200 { // Reasonable upper bound
			t.Errorf("Line %d appears too long (%d chars), may not be wrapped properly: %s", i, len(line), line)
		}
	}
}

// TestChatViewErrorRenderingShortMessage tests that short error messages are handled correctly
func TestChatViewErrorRenderingShortMessage(t *testing.T) {
	chatView := NewChatView(100, 20)

	shortError := "File not found."

	errorMsg := types.ChatMessage{
		Role:      "error",
		Content:   shortError,
		Timestamp: time.Now(),
	}

	rendered := chatView.renderMessage(errorMsg)

	// Should still have proper formatting
	if !strings.Contains(rendered, "⚠") {
		t.Error("Short error message should contain warning icon")
	}

	if !strings.Contains(rendered, shortError) {
		t.Error("Short error message should contain the original content")
	}
}

// TestChatViewErrorRenderingEmptyMessage tests handling of empty error messages
func TestChatViewErrorRenderingEmptyMessage(t *testing.T) {
	chatView := NewChatView(100, 20)

	errorMsg := types.ChatMessage{
		Role:      "error",
		Content:   "",
		Timestamp: time.Now(),
	}

	rendered := chatView.renderMessage(errorMsg)

	// Should still have proper header formatting
	if !strings.Contains(rendered, "⚠") {
		t.Error("Empty error message should still contain warning icon")
	}

	if !strings.Contains(rendered, "Error") {
		t.Error("Empty error message should still contain 'Error' text")
	}
}

// TestChatViewErrorRenderingSpecialCharacters tests that special characters are preserved
func TestChatViewErrorRenderingSpecialCharacters(t *testing.T) {
	chatView := NewChatView(100, 20)

	specialError := "Error: файл не найден ❌🔌 @#$%^&*()"

	errorMsg := types.ChatMessage{
		Role:      "error",
		Content:   specialError,
		Timestamp: time.Now(),
	}

	rendered := chatView.renderMessage(errorMsg)

	// Check that special characters are preserved
	specialChars := []string{"файл", "❌", "🔌", "@#$%^&*()"}
	for _, char := range specialChars {
		if !strings.Contains(rendered, char) {
			t.Errorf("Error message should preserve special character: %s", char)
		}
	}
}

// TestWordWrapFunctionality tests the word wrap function directly
func TestWordWrapFunctionality(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected []string // Expected lines after wrapping
	}{
		{
			name:     "Simple wrap",
			input:    "This is a simple test message",
			width:    10,
			expected: []string{"This is a", "simple", "test", "message"},
		},
		{
			name:     "No wrap needed",
			input:    "Short",
			width:    20,
			expected: []string{"Short"},
		},
		{
			name:     "Single long word",
			input:    "Supercalifragilisticexpialidocious",
			width:    10,
			expected: []string{"Supercalifragilisticexpialidocious"},
		},
		{
			name:     "Multiple lines with wrap",
			input:    "This is a very long error message that should wrap properly across multiple lines",
			width:    25,
			expected: []string{
				"This is a very long error",
				"message that should wrap",
				"properly across multiple",
				"lines",
			},
		},
		{
			name:     "Empty string",
			input:    "",
			width:    10,
			expected: []string{},
		},
		{
			name:     "Zero width",
			input:    "Test message",
			width:    0,
			expected: []string{"Test message"},
		},
		{
			name:     "Negative width",
			input:    "Test message",
			width:    -10,
			expected: []string{"Test message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wordWrap(tt.input, tt.width)
			lines := []string{}
			if result != "" {
				lines = strings.Split(result, "\n")
			}

			if len(lines) != len(tt.expected) {
				t.Errorf("wordWrap(%q, %d) produced %d lines, expected %d lines",
					tt.input, tt.width, len(lines), len(tt.expected))
				t.Errorf("Got: %v", lines)
				t.Errorf("Expected: %v", tt.expected)
				return
			}

			for i, line := range lines {
				if line != tt.expected[i] {
					t.Errorf("wordWrap line %d: got %q, expected %q", i, line, tt.expected[i])
				}
			}
		})
	}
}

// TestWordWrapWidth tests that word wrap respects width limits
func TestWordWrapWidth(t *testing.T) {
	widths := []int{10, 20, 40, 80}
	message := "This is a test message that should be wrapped at different widths to ensure the word wrap function works correctly."

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			result := wordWrap(message, width)
			lines := strings.Split(result, "\n")

			for i, line := range lines {
				if len(line) > width {
					t.Errorf("Line %d exceeds width %d: %q (%d chars)", i, width, line, len(line))
				}
			}
		})
	}
}

// Need to import fmt for the test above
// Let me update the imports at the top