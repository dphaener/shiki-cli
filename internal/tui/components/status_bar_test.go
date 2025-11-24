package components

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderStatusBar_Basic(t *testing.T) {
	result := RenderStatusBar(5, "turns", 100)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Files: 5")
	assert.Contains(t, result, "Pane: turns")
}

func TestRenderStatusBar_FilePane(t *testing.T) {
	result := RenderStatusBar(10, "file", 100)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Files: 10")
	assert.Contains(t, result, "Pane: file")
}

func TestRenderStatusBar_NoFiles(t *testing.T) {
	result := RenderStatusBar(0, "turns", 100)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Files: 0")
}

func TestRenderStatusBar_ManyFiles(t *testing.T) {
	result := RenderStatusBar(999, "turns", 120)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Files: 999")
}

func TestRenderStatusBar_Keybindings(t *testing.T) {
	result := RenderStatusBar(5, "turns", 120)

	assert.NotEmpty(t, result)

	// Verify key bindings are displayed
	assert.Contains(t, result, "Navigate")
	assert.Contains(t, result, "Switch")
	assert.Contains(t, result, "File")
	assert.Contains(t, result, "Quit")
	assert.Contains(t, result, "Help")
}

func TestRenderStatusBar_DifferentWidths(t *testing.T) {
	widths := []int{60, 80, 100, 120, 150}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			result := RenderStatusBar(5, "turns", width)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "Files: 5")
			assert.Contains(t, result, "Pane: turns")
		})
	}
}

func TestRenderStatusBar_NarrowWidth(t *testing.T) {
	// Test with very narrow width
	result := RenderStatusBar(5, "turns", 40)

	assert.NotEmpty(t, result)
	// Should still render without panic
	assert.Contains(t, result, "Files: 5")
}

func TestRenderStatusBar_WideWidth(t *testing.T) {
	// Test with very wide width
	result := RenderStatusBar(5, "turns", 200)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Files: 5")
	assert.Contains(t, result, "Pane: turns")
}

func TestRenderStatusBar_TurnsVsFile(t *testing.T) {
	resultTurns := RenderStatusBar(5, "turns", 100)
	resultFile := RenderStatusBar(5, "file", 100)

	// Both should render
	assert.NotEmpty(t, resultTurns)
	assert.NotEmpty(t, resultFile)

	// Should show different pane names
	assert.Contains(t, resultTurns, "Pane: turns")
	assert.Contains(t, resultFile, "Pane: file")

	// Results should be different
	assert.NotEqual(t, resultTurns, resultFile)
}

func TestRenderStatusBar_ConsistentLayout(t *testing.T) {
	// Test that the status bar maintains consistent layout
	result1 := RenderStatusBar(1, "turns", 100)
	result2 := RenderStatusBar(100, "turns", 100)

	// Both should render
	assert.NotEmpty(t, result1)
	assert.NotEmpty(t, result2)

	// Both should contain key components
	assert.Contains(t, result1, "Files:")
	assert.Contains(t, result2, "Files:")
	assert.Contains(t, result1, "Pane:")
	assert.Contains(t, result2, "Pane:")
}

func TestRenderStatusBar_AllKeybindings(t *testing.T) {
	result := RenderStatusBar(5, "turns", 150)

	// Verify all 5 keybindings are present
	keybindings := []string{
		"Navigate",
		"Switch",
		"File",
		"Quit",
		"Help",
	}

	for _, kb := range keybindings {
		assert.Contains(t, result, kb, "Should contain keybinding: %s", kb)
	}
}

func TestRenderStatusBar_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		fileCount int
		pane      string
		width     int
		panics    bool
	}{
		{"zero width", 0, "turns", 0, false},
		{"negative file count", -1, "turns", 100, false},
		{"empty pane name", 5, "", 100, false},
		{"very large file count", 999999, "turns", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				assert.Panics(t, func() {
					RenderStatusBar(tt.fileCount, tt.pane, tt.width)
				})
			} else {
				assert.NotPanics(t, func() {
					result := RenderStatusBar(tt.fileCount, tt.pane, tt.width)
					assert.NotEmpty(t, result)
				})
			}
		})
	}
}
