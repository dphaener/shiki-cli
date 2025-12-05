package components

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dphaener/shiki-cli/pkg/types"
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

// ===============================================
// Spinner State Management Tests (Task 1.1, 1.2)
// ===============================================

// TestSpinnerHealthValidation tests validateSpinnerHealth() with various timing scenarios
func TestSpinnerHealthValidation(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Test 1: Spinner not active - should always be healthy
	if !chatView.validateSpinnerHealth() {
		t.Error("Spinner should be healthy when not active")
	}

	// Test 2: Loading state active but spinner not active - should be healthy
	chatView.showingLoadingState = true
	chatView.spinnerActive = false
	if !chatView.validateSpinnerHealth() {
		t.Error("Spinner should be healthy when loading state active but spinner not active")
	}

	// Test 3: Spinner active with recent tick - should be healthy
	chatView.spinnerActive = true
	chatView.lastTickTime = time.Now().Add(-50 * time.Millisecond) // Recent tick
	chatView.spinnerTickTimeout = 200 * time.Millisecond
	if !chatView.validateSpinnerHealth() {
		t.Error("Spinner should be healthy with recent tick")
	}

	// Test 4: Spinner active with old tick - should be unhealthy
	chatView.lastTickTime = time.Now().Add(-300 * time.Millisecond) // Old tick
	if chatView.validateSpinnerHealth() {
		t.Error("Spinner should be unhealthy with old tick")
	}

	// Test 5: Edge case - exactly at timeout boundary
	chatView.lastTickTime = time.Now().Add(-200 * time.Millisecond) // Exactly at timeout
	// This might be healthy or unhealthy depending on timing precision
	health := chatView.validateSpinnerHealth()
	_ = health // Just verify it doesn't panic

	// Test 6: Zero time value
	chatView.lastTickTime = time.Time{}
	if chatView.validateSpinnerHealth() {
		t.Error("Spinner should be unhealthy with zero time value")
	}
}

// TestSpinnerTickRestart tests restartSpinnerTick() command generation
func TestSpinnerTickRestart(t *testing.T) {
	chatView := NewChatView(100, 20)

	beforeTime := time.Now()
	cmd := chatView.restartSpinnerTick()
	afterTime := time.Now()

	// Verify command is not nil
	if cmd == nil {
		t.Error("restartSpinnerTick should return a non-nil command")
	}

	// Verify lastTickTime was updated
	if chatView.lastTickTime.Before(beforeTime) || chatView.lastTickTime.After(afterTime) {
		t.Error("restartSpinnerTick should update lastTickTime to current time")
	}

	// Verify command is the spinner tick command (we can't easily test the exact command,
	// but we can verify it's not nil and doesn't panic when called)
	if cmd == nil {
		t.Error("Spinner tick command should not be nil")
	}
}

// TestSpinnerStateInitialization tests proper field initialization
func TestSpinnerStateInitialization(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Verify initial spinner state
	if chatView.spinnerActive {
		t.Error("spinnerActive should be false initially")
	}

	if !chatView.lastTickTime.IsZero() {
		t.Error("lastTickTime should be zero initially")
	}

	if chatView.expectedTickID != 0 {
		t.Error("expectedTickID should be 0 initially")
	}

	expectedTimeout := 200 * time.Millisecond
	if chatView.spinnerTickTimeout != expectedTimeout {
		t.Errorf("spinnerTickTimeout should be %v, got %v", expectedTimeout, chatView.spinnerTickTimeout)
	}
}

// TestSpinnerStateCleanup tests proper field reset
func TestSpinnerStateCleanup(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Set up spinner state
	chatView.SetLoadingState("plan")

	// Verify state is set
	if !chatView.spinnerActive {
		t.Error("spinnerActive should be true after SetLoadingState")
	}

	if chatView.lastTickTime.IsZero() {
		t.Error("lastTickTime should not be zero after SetLoadingState")
	}

	// Clear loading state
	chatView.ClearLoadingState()

	// Verify state is reset
	if chatView.spinnerActive {
		t.Error("spinnerActive should be false after ClearLoadingState")
	}

	if !chatView.lastTickTime.IsZero() {
		t.Error("lastTickTime should be zero after ClearLoadingState")
	}

	if chatView.showingLoadingState {
		t.Error("showingLoadingState should be false after ClearLoadingState")
	}

	if chatView.loadingMessage != "" {
		t.Error("loadingMessage should be empty after ClearLoadingState")
	}
}

// ===========================================
// Update Method Tests (Task 2.1)
// ===========================================

// TestTickMessageHandling verifies TickMsg updates timing fields
func TestTickMessageHandling(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Set spinner active
	chatView.spinnerActive = true

	// Create a TickMsg
	testTime := time.Now()
	testID := 42
	tickMsg := spinner.TickMsg{
		Time: testTime,
		ID:   testID,
	}

	// Update with TickMsg
	updatedChatView, cmd := chatView.Update(tickMsg)

	// Verify timing fields were updated
	if updatedChatView.lastTickTime != testTime {
		t.Errorf("lastTickTime should be updated to %v, got %v", testTime, updatedChatView.lastTickTime)
	}

	if updatedChatView.expectedTickID != testID {
		t.Errorf("expectedTickID should be updated to %d, got %d", testID, updatedChatView.expectedTickID)
	}

	// Note: Command may or may not be returned depending on component state,
	// but the Update method should not panic and should process the message
	_ = cmd // Command return is not guaranteed for all message types
}

// TestTickMessageHandlingInactive verifies TickMsg is ignored when spinner inactive
func TestTickMessageHandlingInactive(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Ensure spinner is inactive
	chatView.spinnerActive = false
	originalTime := chatView.lastTickTime
	originalID := chatView.expectedTickID

	// Create a TickMsg
	tickMsg := spinner.TickMsg{
		Time: time.Now(),
		ID:   42,
	}

	// Update with TickMsg
	updatedChatView, _ := chatView.Update(tickMsg)

	// Verify timing fields were NOT updated
	if updatedChatView.lastTickTime != originalTime {
		t.Error("lastTickTime should not be updated when spinner inactive")
	}

	if updatedChatView.expectedTickID != originalID {
		t.Error("expectedTickID should not be updated when spinner inactive")
	}
}

// TestCommandBatching verifies all commands preserved in batch
func TestCommandBatching(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Create a test message that should generate commands from components
	testMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}

	// Update with test message
	_, cmd := chatView.Update(testMsg)

	// Verify command is returned (exact testing of command content is complex,
	// but we can verify a command is generated)
	if cmd == nil {
		t.Error("Update should return a command when components generate commands")
	}
}

// TestSpinnerRecovery tests automatic recovery triggering
func TestSpinnerRecovery(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Set up unhealthy spinner state
	chatView.showingLoadingState = true
	chatView.spinnerActive = true
	chatView.lastTickTime = time.Now().Add(-500 * time.Millisecond) // Very old tick
	chatView.spinnerTickTimeout = 200 * time.Millisecond

	// Update should trigger recovery
	beforeTime := time.Now()
	updatedChatView, cmd := chatView.Update(tea.KeyMsg{})
	afterTime := time.Now()

	// Verify recovery was triggered (lastTickTime should be updated)
	if updatedChatView.lastTickTime.Before(beforeTime) || updatedChatView.lastTickTime.After(afterTime) {
		t.Error("Spinner recovery should update lastTickTime")
	}

	// Verify command is returned
	if cmd == nil {
		t.Error("Spinner recovery should generate a command")
	}
}

// TestUpdateMethodBackwardCompatibility ensures existing behavior preserved
func TestUpdateMethodBackwardCompatibility(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Test with regular key message
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")}
	updatedChatView, cmd := chatView.Update(keyMsg)

	// Verify basic functionality still works
	if cmd == nil {
		t.Error("Update should still return commands for regular messages")
	}

	// Verify structure is preserved
	if updatedChatView.width != chatView.width || updatedChatView.height != chatView.height {
		t.Error("Update should preserve basic ChatView structure")
	}
}

// ===========================================
// Loading State Tests (Task 3.1, 3.2)
// ===========================================

// TestSetLoadingStateSpinnerInit verifies spinner fields properly initialized
func TestSetLoadingStateSpinnerInit(t *testing.T) {
	chatView := NewChatView(100, 20)

	beforeTime := time.Now()
	chatView.SetLoadingState("plan")
	afterTime := time.Now()

	// Verify spinner fields are initialized
	if !chatView.spinnerActive {
		t.Error("spinnerActive should be true after SetLoadingState")
	}

	if !chatView.showingLoadingState {
		t.Error("showingLoadingState should be true after SetLoadingState")
	}

	if chatView.lastTickTime.Before(beforeTime) || chatView.lastTickTime.After(afterTime) {
		t.Error("lastTickTime should be set to current time")
	}

	if chatView.spinnerTickTimeout != 200*time.Millisecond {
		t.Errorf("spinnerTickTimeout should be 200ms, got %v", chatView.spinnerTickTimeout)
	}

	if chatView.currentPhase != "plan" {
		t.Errorf("currentPhase should be 'plan', got %s", chatView.currentPhase)
	}

	if !chatView.contentDirty {
		t.Error("contentDirty should be true to trigger re-render")
	}
}

// TestClearLoadingStateCleanup verifies spinner fields properly reset
func TestClearLoadingStateCleanup(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Set loading state first
	chatView.SetLoadingState("implement")

	// Verify it's set
	if !chatView.showingLoadingState || !chatView.spinnerActive {
		t.Error("Loading state should be active before clearing")
	}

	// Clear loading state
	chatView.ClearLoadingState()

	// Verify all fields are reset
	if chatView.showingLoadingState {
		t.Error("showingLoadingState should be false after clear")
	}

	if chatView.spinnerActive {
		t.Error("spinnerActive should be false after clear")
	}

	if chatView.loadingMessage != "" {
		t.Error("loadingMessage should be empty after clear")
	}

	if !chatView.lastTickTime.IsZero() {
		t.Error("lastTickTime should be reset to zero after clear")
	}

	if !chatView.contentDirty {
		t.Error("contentDirty should be true to trigger re-render")
	}
}

// TestLoadingStateTransitions tests state transitions preserve spinner health
func TestLoadingStateTransitions(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Test multiple transitions
	phases := []string{"specify", "plan", "tasks", "implement"}

	for _, phase := range phases {
		chatView.SetLoadingState(phase)

		// Verify spinner is healthy after setting loading state
		if !chatView.validateSpinnerHealth() {
			t.Errorf("Spinner should be healthy after setting loading state to %s", phase)
		}

		// Verify phase is set correctly
		if chatView.currentPhase != phase {
			t.Errorf("currentPhase should be %s, got %s", phase, chatView.currentPhase)
		}

		// Clear and verify
		chatView.ClearLoadingState()
		if chatView.showingLoadingState || chatView.spinnerActive {
			t.Error("Loading state should be fully cleared")
		}
	}
}

// TestSpinnerTimeoutConfiguration verifies timeout value is set correctly
func TestSpinnerTimeoutConfiguration(t *testing.T) {
	chatView := NewChatView(100, 20)

	// Initial timeout should be set
	expectedTimeout := 200 * time.Millisecond
	if chatView.spinnerTickTimeout != expectedTimeout {
		t.Errorf("Initial timeout should be %v, got %v", expectedTimeout, chatView.spinnerTickTimeout)
	}

	// SetLoadingState should maintain the timeout
	chatView.SetLoadingState("plan")
	if chatView.spinnerTickTimeout != expectedTimeout {
		t.Errorf("Timeout should remain %v after SetLoadingState, got %v", expectedTimeout, chatView.spinnerTickTimeout)
	}

	// Clearing should not change timeout (it's a configuration value)
	chatView.ClearLoadingState()
	if chatView.spinnerTickTimeout != expectedTimeout {
		t.Errorf("Timeout should remain %v after ClearLoadingState, got %v", expectedTimeout, chatView.spinnerTickTimeout)
	}
}