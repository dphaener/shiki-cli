package components

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderStatusBar_Basic(t *testing.T) {
	result := RenderStatusBar(2, 0, 100, "")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Agents: 2")
	assert.Contains(t, result, "Agent: 1/2")
}

func TestRenderStatusBar_SecondAgent(t *testing.T) {
	result := RenderStatusBar(2, 1, 100, "")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Agents: 2")
	assert.Contains(t, result, "Agent: 2/2")
}

func TestRenderStatusBar_SingleAgent(t *testing.T) {
	result := RenderStatusBar(1, 0, 100, "")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Agents: 1")
	assert.Contains(t, result, "Agent: 1/1")
}

func TestRenderStatusBar_ManyAgents(t *testing.T) {
	result := RenderStatusBar(5, 2, 120, "")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Agents: 5")
	assert.Contains(t, result, "Agent: 3/5")
}

func TestRenderStatusBar_Keybindings(t *testing.T) {
	result := RenderStatusBar(2, 0, 120, "")

	assert.NotEmpty(t, result)

	// Verify key bindings are displayed
	assert.Contains(t, result, "Scroll")
	assert.Contains(t, result, "Switch Agent")
	assert.Contains(t, result, "Quit")
}

func TestRenderStatusBar_DifferentWidths(t *testing.T) {
	widths := []int{60, 80, 100, 120, 150}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			result := RenderStatusBar(2, 0, width, "")
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "Agents: 2")
			assert.Contains(t, result, "Agent: 1/2")
		})
	}
}

func TestRenderStatusBar_NarrowWidth(t *testing.T) {
	// Test with very narrow width
	result := RenderStatusBar(2, 0, 40, "")

	assert.NotEmpty(t, result)
	// Should still render without panic
	assert.Contains(t, result, "Agents: 2")
}

func TestRenderStatusBar_WideWidth(t *testing.T) {
	// Test with very wide width
	result := RenderStatusBar(2, 0, 200, "")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Agents: 2")
	assert.Contains(t, result, "Agent: 1/2")
}

func TestRenderStatusBar_DifferentAgents(t *testing.T) {
	result1 := RenderStatusBar(2, 0, 100, "")
	result2 := RenderStatusBar(2, 1, 100, "")

	// Both should render
	assert.NotEmpty(t, result1)
	assert.NotEmpty(t, result2)

	// Should show different agent indices
	assert.Contains(t, result1, "Agent: 1/2")
	assert.Contains(t, result2, "Agent: 2/2")

	// Results should be different
	assert.NotEqual(t, result1, result2)
}

func TestRenderStatusBar_ConsistentLayout(t *testing.T) {
	// Test that the status bar maintains consistent layout
	result1 := RenderStatusBar(1, 0, 100, "")
	result2 := RenderStatusBar(5, 2, 100, "")

	// Both should render
	assert.NotEmpty(t, result1)
	assert.NotEmpty(t, result2)

	// Both should contain key components
	assert.Contains(t, result1, "Agents:")
	assert.Contains(t, result2, "Agents:")
	assert.Contains(t, result1, "Agent:")
	assert.Contains(t, result2, "Agent:")
}

func TestRenderStatusBar_AllKeybindings(t *testing.T) {
	result := RenderStatusBar(2, 0, 150, "")

	// Verify all keybindings are present
	keybindings := []string{
		"Scroll",
		"Switch Agent",
		"Quit",
	}

	for _, kb := range keybindings {
		assert.Contains(t, result, kb, "Should contain keybinding: %s", kb)
	}
}

func TestRenderStatusBar_WithRecentTool(t *testing.T) {
	result := RenderStatusBar(2, 0, 150, "agent1: Read")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "agent1: Read")
}

func TestRenderStatusBar_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		agentCount int
		agentIndex int
		width      int
		recentTool string
		panics     bool
	}{
		{"zero width", 1, 0, 0, "", false},
		{"single agent", 1, 0, 100, "", false},
		{"many agents", 10, 5, 100, "", false},
		{"with tool activity", 2, 0, 100, "agent: Tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				assert.Panics(t, func() {
					RenderStatusBar(tt.agentCount, tt.agentIndex, tt.width, tt.recentTool)
				})
			} else {
				assert.NotPanics(t, func() {
					result := RenderStatusBar(tt.agentCount, tt.agentIndex, tt.width, tt.recentTool)
					assert.NotEmpty(t, result)
				})
			}
		})
	}
}
