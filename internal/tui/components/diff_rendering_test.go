package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dphaener/shiki-cli/internal/conversation"
	"github.com/dphaener/shiki-cli/internal/diff"
)

func TestRenderToolResultPart_WithDiff(t *testing.T) {
	// Create a sample diff
	generator := diff.NewGenerator()
	fileDiff, err := generator.GenerateUnifiedDiff(
		"old content\nline 2",
		"new content\nline 2\nline 3",
		"test.txt",
	)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	// Create tool result part with diff
	part := conversation.NewToolResultPartWithDiff(
		"test-tool-call",
		"File updated successfully",
		false,
		fileDiff,
	)

	// Test rendering at different widths
	widths := []int{40, 80, 120, 200}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			result := renderToolResultPart(part, width)

			// Basic checks
			if result == "" {
				t.Error("Expected non-empty result")
			}

			// Should contain the tool result prefix
			if !strings.Contains(result, "Result:") {
				t.Error("Expected result to contain 'Result:'")
			}

			// Should contain file path (header)
			if !strings.Contains(result, "test.txt") {
				t.Error("Expected result to contain file path")
			}

			// Should contain diff markers
			hasAddition := strings.Contains(result, "+")
			hasDeletion := strings.Contains(result, "-")

			if !hasAddition || !hasDeletion {
				t.Error("Expected diff to contain both additions and deletions")
			}

			// Check line count isn't excessive
			lines := strings.Split(result, "\n")
			if len(lines) > 100 {
				t.Errorf("Result has too many lines: %d, expected <= 100", len(lines))
			}
		})
	}
}

func TestRenderToolResultPart_WithoutDiff(t *testing.T) {
	// Create tool result part without diff
	part := conversation.NewToolResultPart(
		"test-tool-call",
		"Operation completed",
		false,
	)

	result := renderToolResultPart(part, 80)

	// Should contain basic tool result
	if !strings.Contains(result, "Result:") {
		t.Error("Expected result to contain 'Result:'")
	}

	if !strings.Contains(result, "Operation completed") {
		t.Error("Expected result to contain tool content")
	}

	// Should not contain diff content
	if strings.Contains(result, "📝") {
		t.Error("Should not contain diff header when no diff present")
	}
}

func TestRenderToolResultPart_WithError(t *testing.T) {
	// Create a diff for error case
	generator := diff.NewGenerator()
	fileDiff, _ := generator.GenerateUnifiedDiff(
		"old content",
		"new content",
		"test.txt",
	)

	// Create tool result part with error
	part := conversation.NewToolResultPartWithDiff(
		"test-tool-call",
		"Error: File not found",
		true, // IsError = true
		fileDiff,
	)

	result := renderToolResultPart(part, 80)

	// Should contain error prefix
	if !strings.Contains(result, "Error:") {
		t.Error("Expected result to contain 'Error:'")
	}

	// Should NOT contain diff when there's an error
	if strings.Contains(result, "📝") {
		t.Error("Should not contain diff when tool result is an error")
	}
}

func TestRenderDiff_VariousWidths(t *testing.T) {
	generator := diff.NewGenerator()
	fileDiff, err := generator.GenerateUnifiedDiff(
		"This is a very long line that will test the line wrapping functionality in terminal displays",
		"This is a very long modified line that will test the line wrapping functionality in terminal displays with changes",
		"long-line-test.txt",
	)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	widths := []int{40, 60, 80, 120} // Skip very narrow widths that are impractical

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			result := renderDiff(fileDiff, width)

			if result == "" {
				t.Error("Expected non-empty diff result")
			}

			// Check that lines don't exceed reasonable width
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				// Remove ANSI color codes for length checking
				cleanLine := stripANSI(line)
				if len(cleanLine) > width+10 { // Allow some tolerance for styling
					t.Errorf("Line too long for width %d: %d chars in line: %s", width, len(cleanLine), cleanLine)
				}
			}
		})
	}
}

func TestRenderDiff_BinaryFile(t *testing.T) {
	// Create a binary file diff
	generator := diff.NewGenerator()
	fileDiff, err := generator.GenerateUnifiedDiff(
		"binary content with\x00null bytes",
		"modified binary content with\x00null bytes",
		"binary.dat",
	)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	result := renderDiff(fileDiff, 80)

	// Should contain binary file indicator
	if !strings.Contains(result, "Binary file") {
		t.Error("Expected binary file indicator")
	}

	// Should contain file name
	if !strings.Contains(result, "binary.dat") {
		t.Error("Expected file name in binary diff")
	}
}

func TestRenderDiff_NilDiff(t *testing.T) {
	result := renderDiff(nil, 80)

	if result != "" {
		t.Error("Expected empty result for nil diff")
	}
}

// stripANSI removes ANSI color codes for length testing
func stripANSI(input string) string {
	// Simple regex would be better, but for testing we'll use basic replacement
	cleaned := input
	ansiCodes := []string{
		"\x1b[0m",   // Reset
		"\x1b[1m",   // Bold
		"\x1b[2m",   // Dim
		"\x1b[3m",   // Italic
		"\x1b[4m",   // Underline
		"\x1b[30m",  // Black
		"\x1b[31m",  // Red
		"\x1b[32m",  // Green
		"\x1b[33m",  // Yellow
		"\x1b[34m",  // Blue
		"\x1b[35m",  // Magenta
		"\x1b[36m",  // Cyan
		"\x1b[37m",  // White
		"\x1b[90m",  // Bright Black
		"\x1b[91m",  // Bright Red
		"\x1b[92m",  // Bright Green
		"\x1b[93m",  // Bright Yellow
		"\x1b[94m",  // Bright Blue
		"\x1b[95m",  // Bright Magenta
		"\x1b[96m",  // Bright Cyan
		"\x1b[97m",  // Bright White
	}

	for _, code := range ansiCodes {
		cleaned = strings.ReplaceAll(cleaned, code, "")
	}

	// Remove color codes with parameters like \x1b[38;2;r;g;bm
	for strings.Contains(cleaned, "\x1b[") {
		start := strings.Index(cleaned, "\x1b[")
		if start == -1 {
			break
		}
		end := strings.Index(cleaned[start:], "m")
		if end == -1 {
			break
		}
		cleaned = cleaned[:start] + cleaned[start+end+1:]
	}

	return cleaned
}