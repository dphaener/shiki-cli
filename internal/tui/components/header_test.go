package components

import (
	"fmt"
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRenderHeader_Running(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "test-session-123",
		TaskName:    "Test Task",
		Status:      types.SessionRunning,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 3,
		MaxTurns:    10,
		TotalCost:   0.15,
		TotalTokens: 3000,
	}

	header := RenderHeader(session, 100)

	// Verify key components
	assert.Contains(t, header, "test-ses") // Truncated session ID
	assert.Contains(t, header, "Test Task")
	assert.Contains(t, header, "running")
	assert.Contains(t, header, "3/10") // Current turn
	assert.Contains(t, header, "$0.1500")
	assert.NotEmpty(t, header)
}

func TestRenderHeader_Paused(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "paused-session",
		TaskName:    "Paused Task",
		Status:      types.SessionPaused,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 5,
		MaxTurns:    20,
		TotalCost:   0.50,
	}

	header := RenderHeader(session, 120)

	assert.Contains(t, header, "Paused Task")
	assert.Contains(t, header, "paused")
	assert.Contains(t, header, "5/20")
	assert.Contains(t, header, "$0.5000")
}

func TestRenderHeader_Completed(t *testing.T) {
	now := time.Now()
	completed := now.Add(5 * time.Minute)
	session := &types.Session{
		ID:          "completed-session",
		TaskName:    "Completed Task",
		Status:      types.SessionCompleted,
		CreatedAt:   now,
		StartedAt:   &now,
		CompletedAt: &completed,
		CurrentTurn: 10,
		MaxTurns:    10,
		TotalCost:   1.25,
	}

	header := RenderHeader(session, 120)

	assert.Contains(t, header, "Completed Task")
	assert.Contains(t, header, "completed")
	assert.Contains(t, header, "10/10")
	assert.Contains(t, header, "$1.2500")
	assert.Contains(t, header, "5m") // Duration
}

func TestRenderHeader_Error(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "error-session",
		TaskName:    "Failed Task",
		Status:      types.SessionError,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 2,
		MaxTurns:    10,
		TotalCost:   0.05,
	}

	header := RenderHeader(session, 100)

	assert.Contains(t, header, "Failed Task")
	assert.Contains(t, header, "error")
	assert.Contains(t, header, "2/10")
}

func TestRenderHeader_Incomplete(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "incomplete-session",
		TaskName:    "Incomplete Task",
		Status:      types.SessionIncomplete,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 10,
		MaxTurns:    10,
		TotalCost:   0.75,
	}

	header := RenderHeader(session, 100)

	assert.Contains(t, header, "Incomplete Task")
	assert.Contains(t, header, "incomplete")
}

func TestRenderHeader_NilSession(t *testing.T) {
	header := RenderHeader(nil, 100)
	assert.Empty(t, header)
}

func TestRenderHeader_DifferentWidths(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "test-session-123", // Use 8+ chars
		TaskName:    "Test",
		Status:      types.SessionRunning,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 1,
		MaxTurns:    10,
		TotalCost:   0.10,
	}

	widths := []int{60, 80, 100, 120, 150}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			header := RenderHeader(session, width)
			assert.NotEmpty(t, header)
			// Header should adapt to different widths
			assert.Contains(t, header, "Test")
		})
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   string
		contains string // What the styled output should contain
	}{
		{"running", "running"},
		{"paused", "paused"},
		{"completed", "completed"},
		{"error", "error"},
		{"incomplete", "incomplete"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := formatStatus(tt.status)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5 * time.Minute, "5m0s"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m30s"},
		{"hours", 2 * time.Hour, "2h0m0s"},
		{"hours minutes seconds", 2*time.Hour + 15*time.Minute + 30*time.Second, "2h15m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderHeader_SessionIDTruncation(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "very-long-session-id-12345678900", // > 8 chars
		TaskName:    "Test",
		Status:      types.SessionRunning,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 1,
		MaxTurns:    10,
	}

	header := RenderHeader(session, 100)

	// Should truncate to first 8 characters
	assert.Contains(t, header, "very-lon")
	assert.NotContains(t, header, "12345678900")
}

func TestRenderHeader_ZeroCost(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "test-session-123",
		TaskName:    "Test",
		Status:      types.SessionRunning,
		CreatedAt:   now,
		StartedAt:   &now,
		CurrentTurn: 1,
		MaxTurns:    10,
		TotalCost:   0.0,
	}

	header := RenderHeader(session, 100)

	assert.Contains(t, header, "$0.0000")
}

func TestRenderHeader_NoStartTime(t *testing.T) {
	now := time.Now()
	session := &types.Session{
		ID:          "test-session-123",
		TaskName:    "Test",
		Status:      types.SessionRunning, // Use valid status
		CreatedAt:   now,
		StartedAt:   nil, // Not started yet
		CurrentTurn: 0,
		MaxTurns:    10,
		TotalCost:   0.0,
	}

	header := RenderHeader(session, 100)

	// Should still render without errors
	assert.NotEmpty(t, header)
	assert.Contains(t, header, "Test")
	assert.Contains(t, header, "0s") // Duration should be 0
}
