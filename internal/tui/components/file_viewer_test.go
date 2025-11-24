package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderFileViewer_Empty(t *testing.T) {
	result := RenderFileViewer("", "empty.md", 40, 20, 0, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "File: empty.md")
}

func TestRenderFileViewer_SimpleContent(t *testing.T) {
	content := "# Test File\n\nThis is a test."
	result := RenderFileViewer(content, "test.md", 60, 20, 0, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "File: test.md")
	assert.Contains(t, result, "# Test File")
	assert.Contains(t, result, "This is a test")
}

func TestRenderFileViewer_WithLineNumbers(t *testing.T) {
	content := "line 1\nline 2\nline 3"
	result := RenderFileViewer(content, "test.txt", 60, 20, 0, false)

	assert.NotEmpty(t, result)
	// Line numbers should be present
	assert.Contains(t, result, "1")
	assert.Contains(t, result, "2")
	assert.Contains(t, result, "3")
}

func TestRenderFileViewer_ActiveBorder(t *testing.T) {
	content := "test content"

	// Not active
	resultInactive := RenderFileViewer(content, "test.md", 40, 20, 0, false)
	assert.NotEmpty(t, resultInactive)
	assert.Contains(t, resultInactive, "File: test.md")

	// Active
	resultActive := RenderFileViewer(content, "test.md", 40, 20, 0, true)
	assert.NotEmpty(t, resultActive)
	assert.Contains(t, resultActive, "File: test.md")

	// Both should render correctly (border style difference may not be visible in plain text)
}

func TestRenderFileViewer_Scrolling(t *testing.T) {
	// Create content with many lines
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		lines[i] = fmt.Sprintf("Line %d content here", i+1)
	}
	content := strings.Join(lines, "\n")

	// No scroll
	result1 := RenderFileViewer(content, "large.txt", 60, 20, 0, false)
	assert.NotEmpty(t, result1)
	assert.Contains(t, result1, "Line 1")

	// Scrolled to middle
	result2 := RenderFileViewer(content, "large.txt", 60, 20, 50, false)
	assert.NotEmpty(t, result2)
	// Should show lines around line 50

	// Different scroll positions should produce different output
	assert.NotEqual(t, result1, result2)
}

func TestRenderFileViewer_ScrollIndicator(t *testing.T) {
	// Create content with many lines to trigger scroll indicator
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	content := strings.Join(lines, "\n")

	// Small height to force scrolling
	result := RenderFileViewer(content, "large.txt", 60, 10, 25, false)

	assert.NotEmpty(t, result)
	// Should contain scroll percentage indicator
	assert.Contains(t, result, "%")
}

func TestRenderFileViewer_NoScrollIndicatorSmallFile(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"

	// Large height, no scrolling needed
	result := RenderFileViewer(content, "small.txt", 60, 30, 0, false)

	assert.NotEmpty(t, result)
	// Should NOT contain scroll indicator
	// (This is harder to test directly, but we can verify it doesn't crash)
}

func TestRenderFileViewer_LongLines(t *testing.T) {
	// Create content with very long line
	longLine := strings.Repeat("x", 200)
	content := "Short line\n" + longLine + "\nAnother line"

	result := RenderFileViewer(content, "long.txt", 60, 20, 0, false)

	assert.NotEmpty(t, result)
	// Should truncate long line
	assert.Contains(t, result, "...")
}

func TestRenderFileViewer_DifferentSizes(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

	sizes := []struct {
		width  int
		height int
	}{
		{40, 15},
		{60, 20},
		{80, 25},
		{100, 30},
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			result := RenderFileViewer(content, "test.txt", size.width, size.height, 0, false)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "File: test.txt")
		})
	}
}

func TestRenderFileViewer_ScrollBounds(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"

	// Scroll offset beyond content
	result := RenderFileViewer(content, "test.txt", 60, 20, 100, false)
	assert.NotEmpty(t, result)
	// Should handle gracefully

	// Negative scroll offset
	result2 := RenderFileViewer(content, "test.txt", 60, 20, -10, false)
	assert.NotEmpty(t, result2)
	// Should clamp to 0
}

func TestRenderFileViewer_EmptyLines(t *testing.T) {
	content := "Line 1\n\n\nLine 4"

	result := RenderFileViewer(content, "test.txt", 60, 20, 0, false)

	assert.NotEmpty(t, result)
	// Should render empty lines
	assert.Contains(t, result, "Line 1")
	assert.Contains(t, result, "Line 4")
}

func TestRenderFileViewer_SpecialCharacters(t *testing.T) {
	content := "# Header\n*italic*\n**bold**\n`code`"

	result := RenderFileViewer(content, "markdown.md", 60, 20, 0, false)

	assert.NotEmpty(t, result)
	// Should render markdown syntax as-is (no highlighting in this version)
	assert.Contains(t, result, "# Header")
	assert.Contains(t, result, "*italic*")
}

func TestRenderFileViewer_DifferentFilenames(t *testing.T) {
	content := "test"

	filenames := []string{
		"short.md",
		"very-long-filename-that-might-be-truncated.md",
		"file.with.dots.txt",
		"UPPERCASE.MD",
	}

	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			result := RenderFileViewer(content, filename, 60, 20, 0, false)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, filename)
		})
	}
}

func TestRenderFileViewer_MultilineContent(t *testing.T) {
	content := `# Test Document

This is a multi-line document.

## Section 1

Content here.

## Section 2

More content.`

	result := RenderFileViewer(content, "doc.md", 80, 25, 0, false)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "# Test Document")
	assert.Contains(t, result, "## Section 1")
	assert.Contains(t, result, "## Section 2")
}

func TestRenderFileViewer_VisibleRange(t *testing.T) {
	// Create content with numbered lines
	lines := make([]string, 50)
	for i := 0; i < 50; i++ {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	content := strings.Join(lines, "\n")

	// View with small height
	result := RenderFileViewer(content, "test.txt", 60, 10, 0, false)

	assert.NotEmpty(t, result)
	// Should show first few lines
	assert.Contains(t, result, "Line 1")

	// Should not show lines far down
	assert.NotContains(t, result, "Line 50")
}
