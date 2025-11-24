package components

import (
	"fmt"
	"testing"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRenderTurnList_Empty(t *testing.T) {
	turns := []types.Turn{}
	result := RenderTurnList(turns, 0, 40, 20, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Turn History")
	assert.Contains(t, result, "No turns yet")
}

func TestRenderTurnList_SingleTurn(t *testing.T) {
	turns := []types.Turn{
		{
			Number:     1,
			AgentID:    "agent1",
			Status:     types.TurnCompleted,
			TokensUsed: 1000,
			Cost:       0.05,
			DurationMs: 1500,
		},
	}

	result := RenderTurnList(turns, 0, 40, 20, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Turn History")
	assert.Contains(t, result, "T01") // Turn number
	assert.Contains(t, result, "agent1")
}

func TestRenderTurnList_MultipleTurns(t *testing.T) {
	turns := []types.Turn{
		{Number: 1, AgentID: "agent1", Status: types.TurnCompleted, Cost: 0.01, DurationMs: 1000},
		{Number: 2, AgentID: "agent2", Status: types.TurnCompleted, Cost: 0.02, DurationMs: 2000},
		{Number: 3, AgentID: "agent1", Status: types.TurnInProgress, Cost: 0.0, DurationMs: 500},
	}

	result := RenderTurnList(turns, 1, 40, 20, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "T01")
	assert.Contains(t, result, "T02")
	assert.Contains(t, result, "T03")
}

func TestRenderTurnList_ActiveBorder(t *testing.T) {
	turns := []types.Turn{
		{Number: 1, AgentID: "agent1", Status: types.TurnCompleted},
	}

	// Not active
	resultInactive := RenderTurnList(turns, 0, 40, 20, false)
	assert.NotEmpty(t, resultInactive)
	assert.Contains(t, resultInactive, "Turn History")

	// Active
	resultActive := RenderTurnList(turns, 0, 40, 20, true)
	assert.NotEmpty(t, resultActive)
	assert.Contains(t, resultActive, "Turn History")

	// Both should render correctly (border style difference may not be visible in plain text)
}

func TestRenderTurnList_Scrolling(t *testing.T) {
	// Create many turns to test scrolling
	var turns []types.Turn
	for i := 1; i <= 50; i++ {
		turns = append(turns, types.Turn{
			Number:     i,
			AgentID:    "agent1",
			Status:     types.TurnCompleted,
			Cost:       0.01,
			DurationMs: 1000,
		})
	}

	// Render with small height to force scrolling
	result := RenderTurnList(turns, 25, 40, 10, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Turn History")

	// Should handle scrolling without errors
	// The view should center around the selected turn
}

func TestRenderTurnList_SelectedTurn(t *testing.T) {
	turns := []types.Turn{
		{Number: 1, AgentID: "agent1", Status: types.TurnCompleted},
		{Number: 2, AgentID: "agent2", Status: types.TurnCompleted},
		{Number: 3, AgentID: "agent1", Status: types.TurnCompleted},
	}

	// Select first turn
	result1 := RenderTurnList(turns, 0, 40, 20, false)
	assert.NotEmpty(t, result1)
	assert.Contains(t, result1, "T01")

	// Select last turn
	result2 := RenderTurnList(turns, 2, 40, 20, false)
	assert.NotEmpty(t, result2)
	assert.Contains(t, result2, "T03")

	// Both should render all turns
	assert.Contains(t, result1, "T01")
	assert.Contains(t, result2, "T01")
}

func TestRenderTurnList_DifferentSizes(t *testing.T) {
	turns := []types.Turn{
		{Number: 1, AgentID: "agent1", Status: types.TurnCompleted, Cost: 0.01, DurationMs: 1000},
	}

	sizes := []struct {
		width  int
		height int
	}{
		{30, 10},
		{40, 20},
		{60, 30},
		{80, 40},
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			result := RenderTurnList(turns, 0, size.width, size.height, false)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "Turn History")
		})
	}
}

func TestFormatTurnLine_Completed(t *testing.T) {
	turn := types.Turn{
		Number:     5,
		AgentID:    "test-agent",
		Status:     types.TurnCompleted,
		TokensUsed: 1500,
		Cost:       0.075,
		DurationMs: 2500,
	}

	line := formatTurnLine(turn, false, 50)

	assert.Contains(t, line, "T05")
	assert.Contains(t, line, "test-agent")
	assert.Contains(t, line, "$0.075")
	assert.Contains(t, line, "2.5s") // Duration formatting
}

func TestFormatTurnLine_InProgress(t *testing.T) {
	turn := types.Turn{
		Number:  3,
		AgentID: "agent1",
		Status:  types.TurnInProgress,
	}

	line := formatTurnLine(turn, false, 50)

	assert.Contains(t, line, "T03")
	assert.Contains(t, line, "agent1")
	// In progress status indicator should be present
}

func TestFormatTurnLine_Error(t *testing.T) {
	turn := types.Turn{
		Number:  2,
		AgentID: "agent2",
		Status:  types.TurnError,
	}

	line := formatTurnLine(turn, false, 50)

	assert.Contains(t, line, "T02")
	assert.Contains(t, line, "agent2")
	// Error status indicator should be present
}

func TestFormatTurnLine_Timeout(t *testing.T) {
	turn := types.Turn{
		Number:  4,
		AgentID: "agent1",
		Status:  types.TurnTimeout,
	}

	line := formatTurnLine(turn, false, 50)

	assert.Contains(t, line, "T04")
	assert.Contains(t, line, "agent1")
	// Timeout status indicator should be present
}

func TestFormatTurnLine_Selected(t *testing.T) {
	turn := types.Turn{
		Number:  1,
		AgentID: "agent1",
		Status:  types.TurnCompleted,
	}

	lineUnselected := formatTurnLine(turn, false, 50)
	lineSelected := formatTurnLine(turn, true, 50)

	// Both should contain turn info
	assert.Contains(t, lineUnselected, "T01")
	assert.Contains(t, lineSelected, "T01")
	assert.Contains(t, lineUnselected, "agent1")
	assert.Contains(t, lineSelected, "agent1")

	// Both should be valid rendered lines
	assert.NotEmpty(t, lineUnselected)
	assert.NotEmpty(t, lineSelected)
}

func TestFormatTurnLine_LongAgentID(t *testing.T) {
	turn := types.Turn{
		Number:  1,
		AgentID: "very-long-agent-id-that-exceeds-limit",
		Status:  types.TurnCompleted,
	}

	line := formatTurnLine(turn, false, 50)

	// Should truncate agent ID
	assert.Contains(t, line, "very-long-ag") // First 12 chars
	assert.NotContains(t, line, "that-exceeds-limit")
}

func TestFormatTurnLine_DurationFormatting(t *testing.T) {
	tests := []struct {
		name       string
		durationMS int64
		contains   string
	}{
		{"milliseconds", 500, "500ms"},
		{"one second", 1000, "1.0s"},
		{"multiple seconds", 3500, "3.5s"},
		{"very fast", 50, "50ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := types.Turn{
				Number:     1,
				AgentID:    "agent1",
				Status:     types.TurnCompleted,
				DurationMs: tt.durationMS,
			}

			line := formatTurnLine(turn, false, 50)
			assert.Contains(t, line, tt.contains)
		})
	}
}

func TestFormatTurnLine_CostFormatting(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		contains string
	}{
		{"zero cost", 0.0, "$0.000"},
		{"small cost", 0.001, "$0.001"},
		{"medium cost", 0.05, "$0.050"},
		{"large cost", 1.234, "$1.234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := types.Turn{
				Number:  1,
				AgentID: "agent1",
				Status:  types.TurnCompleted,
				Cost:    tt.cost,
			}

			line := formatTurnLine(turn, false, 50)
			assert.Contains(t, line, tt.contains)
		})
	}
}
