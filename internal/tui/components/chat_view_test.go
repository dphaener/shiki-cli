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

// TestLoadingStatePersistence tests that loading state persists during assistant messages
// and only gets cleared when explicitly called
func TestLoadingStatePersistence(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Set loading state (simulates user sending a message)
	chatView.SetLoadingState("plan")

	// Verify loading state is active
	if !chatView.showingLoadingState {
		t.Error("Loading state should be active after SetLoadingState")
	}

	// Add assistant message (this used to clear loading state prematurely)
	chatView.AddOrUpdateAssistantMessage("Here's my response", true)

	// Loading state should still be active - the fix ensures it doesn't get cleared here
	if !chatView.showingLoadingState {
		t.Error("Loading state should persist after adding assistant message")
	}

	// Add more assistant content (simulating streaming)
	chatView.AddOrUpdateAssistantMessage("Here's my response with more content", true)

	// Loading state should still be active
	if !chatView.showingLoadingState {
		t.Error("Loading state should persist after updating assistant message")
	}

	// Add a tool call (would happen during agent processing)
	toolMsg := types.ChatMessage{
		Role:      "assistant",
		Content:   `Using tool: file_search with {"pattern": "*.go"}`,
		Timestamp: time.Now(),
	}
	chatView.AddToolUse(toolMsg)

	// Loading state should still be active
	if !chatView.showingLoadingState {
		t.Error("Loading state should persist after adding tool use")
	}

	// Only when we explicitly clear loading state should it be removed
	chatView.ClearLoadingState()

	// Now loading state should be cleared
	if chatView.showingLoadingState {
		t.Error("Loading state should be cleared after ClearLoadingState")
	}
}

// TestLoadingStateRendering tests that spinner appears in rendered output when loading
func TestLoadingStateRendering(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Add a regular message first
	chatView.AddOrUpdateAssistantMessage("Regular message", false)

	// Render without loading state
	rendered := chatView.renderMessages()
	if strings.Contains(rendered, "⠋") || strings.Contains(rendered, "⠙") || strings.Contains(rendered, "⠹") {
		t.Error("Should not show spinner when loading state is inactive")
	}

	// Set loading state
	chatView.SetLoadingState("plan")

	// Render with loading state
	rendered = chatView.renderMessages()

	// Should contain loading message
	if !strings.Contains(rendered, "Creating implementation plan") {
		t.Error("Should show loading message when loading state is active")
	}

	// The spinner character itself may vary due to animation, but the general structure should be there
	if !strings.Contains(rendered, "Creating") {
		t.Error("Should contain the loading message text")
	}
}

// TestChatView_ResetScrollState tests the ResetScrollState method functionality
func TestChatView_ResetScrollState(t *testing.T) {
	cv := NewChatView(80, 24)

	// Add some messages to simulate a conversation
	cv.AddMessage(types.ChatMessage{
		Role:      "user",
		Content:   "Test message 1",
		Timestamp: time.Now(),
	})
	cv.AddMessage(types.ChatMessage{
		Role:      "assistant",
		Content:   "Test response 1",
		Timestamp: time.Now(),
	})

	// Simulate some corrupted state
	cv.isStreaming = true
	cv.showingLoadingState = true
	cv.contentDirty = true

	// Reset should fix all state
	cv.ResetScrollState()

	// Verify state is clean
	if cv.isStreaming {
		t.Error("isStreaming should be false after ResetScrollState")
	}
	if cv.showingLoadingState {
		t.Error("showingLoadingState should be false after ResetScrollState")
	}
	if cv.contentDirty {
		t.Error("contentDirty should be false after ResetScrollState")
	}

	// Verify viewport state is reset
	if cv.viewport.YPosition != 0 {
		t.Errorf("viewport.YPosition should be 0, got %d", cv.viewport.YPosition)
	}

	// Verify viewport dimensions are preserved
	expectedHeight := cv.height - 1 - maxInputHeight
	if expectedHeight < 3 {
		expectedHeight = 3
	}
	if cv.viewport.Height != expectedHeight {
		t.Errorf("viewport height should be %d, got %d", expectedHeight, cv.viewport.Height)
	}
	if cv.viewport.Width != cv.width-4 {
		t.Errorf("viewport width should be %d, got %d", cv.width-4, cv.viewport.Width)
	}
}

// TestChatView_ResetScrollStateEmptyMessages tests reset with empty message list
func TestChatView_ResetScrollStateEmptyMessages(t *testing.T) {
	cv := NewChatView(80, 24)

	// Reset with no messages should not panic
	cv.ResetScrollState()

	// Verify clean state
	if cv.isStreaming {
		t.Error("isStreaming should be false after ResetScrollState")
	}
	if cv.showingLoadingState {
		t.Error("showingLoadingState should be false after ResetScrollState")
	}
	if cv.contentDirty {
		t.Error("contentDirty should be false after ResetScrollState")
	}
}

// TestChatView_ResetScrollStateMultipleCalls tests that reset is safe to call multiple times
func TestChatView_ResetScrollStateMultipleCalls(t *testing.T) {
	cv := NewChatView(80, 24)

	// Add a message
	cv.AddMessage(types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	})

	// Reset multiple times should be safe
	cv.ResetScrollState()
	cv.ResetScrollState()
	cv.ResetScrollState()

	// State should still be clean
	if cv.isStreaming || cv.showingLoadingState || cv.contentDirty {
		t.Error("Multiple resets should maintain clean state")
	}
}

// TestChatView_ResetToBottom tests the ResetToBottom method
func TestChatView_ResetToBottom(t *testing.T) {
	cv := NewChatView(80, 24)

	// Add messages
	for i := 0; i < 10; i++ {
		cv.AddMessage(types.ChatMessage{
			Role:      "user",
			Content:   fmt.Sprintf("Test message %d", i),
			Timestamp: time.Now(),
		})
	}

	// ResetToBottom should set contentDirty and go to bottom
	cv.ResetToBottom()

	if !cv.contentDirty {
		t.Error("contentDirty should be true after ResetToBottom")
	}

	// Viewport should be at bottom (this is handled by the GotoBottom call internally)
	// We can't easily test the exact position without more viewport internals,
	// but we can verify the method doesn't panic
}

// TestAgentOutputState_ResetScrollState tests AgentOutputState reset functionality
func TestAgentOutputState_ResetScrollState(t *testing.T) {
	state := &AgentOutputState{
		ScrollOffset: 50,
		AutoScroll:   false,
	}

	// Reset should clean state
	state.ResetScrollState()

	if state.ScrollOffset != 0 {
		t.Errorf("ScrollOffset should be 0 after reset, got %d", state.ScrollOffset)
	}
	if !state.AutoScroll {
		t.Error("AutoScroll should be true after reset")
	}
}

// TestAgentOutputState_ValidateScrollState tests scroll state validation
func TestAgentOutputState_ValidateScrollState(t *testing.T) {
	tests := []struct {
		name               string
		initialOffset      int
		initialAutoScroll  bool
		totalLines         int
		height             int
		expectedOffset     int
		expectedAutoScroll bool
	}{
		{
			name:               "Valid state - no changes",
			initialOffset:      5,
			initialAutoScroll:  false,
			totalLines:         100,
			height:             20,
			expectedOffset:     5,
			expectedAutoScroll: false,
		},
		{
			name:               "Negative offset correction",
			initialOffset:      -10,
			initialAutoScroll:  false,
			totalLines:         100,
			height:             20,
			expectedOffset:     0,
			expectedAutoScroll: false, // At top, autoscroll should remain disabled
		},
		{
			name:               "Offset too high correction",
			initialOffset:      999,
			initialAutoScroll:  false,
			totalLines:         100,
			height:             20,
			expectedOffset:     80, // 100 - 20
			expectedAutoScroll: true, // At bottom, so autoscroll enabled
		},
		{
			name:               "At bottom - enable autoscroll",
			initialOffset:      80,
			initialAutoScroll:  false,
			totalLines:         100,
			height:             20,
			expectedOffset:     80,
			expectedAutoScroll: true,
		},
		{
			name:               "Content smaller than height",
			initialOffset:      10,
			initialAutoScroll:  false,
			totalLines:         5,
			height:             20,
			expectedOffset:     0,
			expectedAutoScroll: true, // Always at bottom when content is small
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &AgentOutputState{
				ScrollOffset: tt.initialOffset,
				AutoScroll:   tt.initialAutoScroll,
			}

			state.ValidateScrollState(tt.totalLines, tt.height)

			if state.ScrollOffset != tt.expectedOffset {
				t.Errorf("ScrollOffset = %d, expected %d", state.ScrollOffset, tt.expectedOffset)
			}
			if state.AutoScroll != tt.expectedAutoScroll {
				t.Errorf("AutoScroll = %t, expected %t", state.AutoScroll, tt.expectedAutoScroll)
			}
		})
	}
}

// TestAgentOutputState_ValidateScrollStateIdempotent tests that validation is idempotent
func TestAgentOutputState_ValidateScrollStateIdempotent(t *testing.T) {
	state := &AgentOutputState{
		ScrollOffset: 999, // Invalid offset
		AutoScroll:   false,
	}

	// Validate multiple times
	state.ValidateScrollState(100, 20)
	firstOffset := state.ScrollOffset
	firstAutoScroll := state.AutoScroll

	state.ValidateScrollState(100, 20)
	secondOffset := state.ScrollOffset
	secondAutoScroll := state.AutoScroll

	// Should be identical
	if firstOffset != secondOffset {
		t.Errorf("Validation not idempotent: offset changed from %d to %d", firstOffset, secondOffset)
	}
	if firstAutoScroll != secondAutoScroll {
		t.Errorf("Validation not idempotent: autoscroll changed from %t to %t", firstAutoScroll, secondAutoScroll)
	}
}