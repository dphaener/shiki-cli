package components

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSekkeiStyleEnumeration(t *testing.T) {
	style := sekkeiStyle()

	// Test that Enumeration field exists and is configured
	assert.NotEmpty(t, style.Enumeration.BlockPrefix, "Enumeration BlockPrefix should not be empty")
	assert.Contains(t, style.Enumeration.BlockPrefix, ". ", "Enumeration should include proper period and space")

	// Verify the exact format matches expectation
	assert.Equal(t, ". ", style.Enumeration.BlockPrefix, "Enumeration BlockPrefix should match expected format")
}

func TestSekkeiStyleItemConfiguration(t *testing.T) {
	style := sekkeiStyle()

	// Test that Item field (unordered lists) is still properly configured
	assert.NotEmpty(t, style.Item.BlockPrefix, "Item BlockPrefix should not be empty")
	assert.Equal(t, "• ", style.Item.BlockPrefix, "Item should use bullet point prefix")
}

func TestSekkeiStyleListConfiguration(t *testing.T) {
	style := sekkeiStyle()

	// Test that List configuration is properly set
	assert.Equal(t, uint(2), style.List.LevelIndent, "List should have proper indentation")
}

func TestMarkdownRendererBasicFunctionality(t *testing.T) {
	renderer := GetMarkdownRenderer()
	assert.NotNil(t, renderer, "GetMarkdownRenderer should return a valid renderer")

	// Test basic rendering
	result, err := renderer.Render("Hello **world**", 80)
	assert.NoError(t, err, "Basic rendering should not produce errors")
	assert.NotEmpty(t, result, "Rendered output should not be empty")
}

func TestMarkdownRendererOrderedList(t *testing.T) {
	renderer := GetMarkdownRenderer()

	// Test simple ordered list
	input := "1. First item\n2. Second item\n3. Third item"
	result, err := renderer.Render(input, 80)

	assert.NoError(t, err, "Ordered list rendering should not produce errors")
	assert.NotEmpty(t, result, "Rendered output should not be empty")

	// Check that the result contains proper numbering
	// Note: The exact format may depend on how glamour processes the template
	assert.Contains(t, result, "1.", "Should contain first item number")
	assert.Contains(t, result, "2.", "Should contain second item number")
	assert.Contains(t, result, "3.", "Should contain third item number")
	assert.Contains(t, result, "First item", "Should contain first item text")
	assert.Contains(t, result, "Second item", "Should contain second item text")
	assert.Contains(t, result, "Third item", "Should contain third item text")
}

func TestMarkdownRendererUnorderedList(t *testing.T) {
	renderer := GetMarkdownRenderer()

	// Test unordered list to ensure no regression
	input := "- First item\n- Second item\n- Third item"
	result, err := renderer.Render(input, 80)

	assert.NoError(t, err, "Unordered list rendering should not produce errors")
	assert.NotEmpty(t, result, "Rendered output should not be empty")

	// Check that bullet points are preserved
	assert.Contains(t, result, "•", "Should contain bullet points")
	assert.Contains(t, result, "First item", "Should contain first item text")
	assert.Contains(t, result, "Second item", "Should contain second item text")
	assert.Contains(t, result, "Third item", "Should contain third item text")
}

// Integration tests for ordered list rendering as specified in Task 3
func TestMarkdownOrderedListRendering(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		expectedPattern string
	}{
		{
			name:            "simple ordered list",
			input:           "1. First item\n2. Second item\n3. Third item",
			expectedPattern: "(?s)1\\. First item.*2\\. Second item.*3\\. Third item",
		},
		{
			name:            "nested ordered list",
			input:           "1. Parent\n   1. Child\n   2. Child 2\n2. Parent 2",
			expectedPattern: "(?s)1\\. Parent.*1\\. Child.*2\\. Child 2.*2\\. Parent 2",
		},
		{
			name:            "mixed numbered start",
			input:           "5. Fifth item\n6. Sixth item\n7. Seventh item",
			expectedPattern: "(?s)5\\. Fifth item.*6\\. Sixth item.*7\\. Seventh item",
		},
		{
			name:            "single item list",
			input:           "1. Only item",
			expectedPattern: "1\\. Only item",
		},
	}

	renderer := GetMarkdownRenderer()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := renderer.Render(tc.input, 80)
			assert.NoError(t, err, "Rendering should not produce errors")
			assert.NotEmpty(t, result, "Rendered output should not be empty")

			// Use regex to match the pattern allowing for any whitespace/formatting
			matched, err := regexp.MatchString(tc.expectedPattern, result)
			assert.NoError(t, err, "Regex pattern should be valid")
			assert.True(t, matched, "Output should match expected pattern.\nExpected pattern: %s\nActual output: %q", tc.expectedPattern, result)
		})
	}
}

func TestMarkdownMixedListTypes(t *testing.T) {
	renderer := GetMarkdownRenderer()

	// Test mixing ordered and unordered lists
	input := "1. Ordered item\n- Unordered item\n2. Another ordered\n  - Nested unordered\n3. Final ordered"
	result, err := renderer.Render(input, 80)

	assert.NoError(t, err, "Mixed list rendering should not produce errors")
	assert.NotEmpty(t, result, "Rendered output should not be empty")

	// Check that both numbering and bullets appear
	assert.Contains(t, result, "1. ", "Should contain ordered list numbering")
	assert.Contains(t, result, "2. ", "Should contain second ordered item")
	assert.Contains(t, result, "3. ", "Should contain third ordered item")
	assert.Contains(t, result, "• ", "Should contain bullet points for unordered items")
}

func TestMarkdownOrderedListWithFormatting(t *testing.T) {
	renderer := GetMarkdownRenderer()

	// Test ordered lists with text formatting
	input := "1. **Bold** item\n2. *Italic* item\n3. `Code` item\n4. [Link](https://example.com) item"
	result, err := renderer.Render(input, 80)

	assert.NoError(t, err, "Formatted list rendering should not produce errors")
	assert.NotEmpty(t, result, "Rendered output should not be empty")

	// Check that numbering is preserved with formatting
	assert.Contains(t, result, "1. ", "Should contain first item numbering")
	assert.Contains(t, result, "2. ", "Should contain second item numbering")
	assert.Contains(t, result, "3. ", "Should contain third item numbering")
	assert.Contains(t, result, "4. ", "Should contain fourth item numbering")
}

// Task 4: Analyze current whitespace trimming behavior
func TestWhitespaceHandling(t *testing.T) {
	renderer := GetMarkdownRenderer()

	testCases := []struct {
		name                   string
		input                  string
		checkTrailingNewlines  bool
		checkListSpacing       bool
		expectedToEndWithSpace bool
	}{
		{
			name:                   "ordered list with trailing newlines",
			input:                  "1. First item\n2. Second item\n\n",
			checkListSpacing:       true,
			checkTrailingNewlines:  true,
			expectedToEndWithSpace: false, // TrimSpace should remove trailing newlines
		},
		{
			name:                   "mixed content with spacing",
			input:                  "Some text\n\n1. List item\n2. Another item\n\nMore text",
			checkListSpacing:       true,
			expectedToEndWithSpace: false,
		},
		{
			name:                   "ordered list only",
			input:                  "1. First item\n2. Second item\n3. Third item",
			checkListSpacing:       true,
			expectedToEndWithSpace: false,
		},
		{
			name:                   "text with leading/trailing whitespace",
			input:                  "   Some text with spaces   ",
			expectedToEndWithSpace: false, // Should be trimmed
		},
		{
			name:                   "list with extra spacing between items",
			input:                  "1. First item\n\n2. Second item\n\n3. Third item",
			checkListSpacing:       true,
			expectedToEndWithSpace: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := renderer.Render(tc.input, 80)
			assert.NoError(t, err, "Rendering should not produce errors")

			if tc.checkTrailingNewlines {
				// Check if trailing newlines are properly handled
				if tc.expectedToEndWithSpace {
					assert.True(t, strings.HasSuffix(result, " ") || strings.HasSuffix(result, "\n"),
						"Should preserve some trailing whitespace")
				} else {
					assert.False(t, strings.HasSuffix(result, "\n\n"),
						"Should not have double trailing newlines")
					assert.False(t, strings.HasSuffix(result, "  "),
						"Should not have trailing spaces")
				}
			}

			if tc.checkListSpacing {
				// Verify that list formatting is preserved
				assert.Contains(t, result, "1. ", "Should contain proper list numbering")
				if strings.Contains(tc.input, "2. ") {
					assert.Contains(t, result, "2. ", "Should contain second item numbering")
				}
				if strings.Contains(tc.input, "3. ") {
					assert.Contains(t, result, "3. ", "Should contain third item numbering")
				}

				// Check that items are properly separated
				lines := strings.Split(result, "\n")
				listLines := make([]string, 0)
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if strings.Contains(trimmed, ". ") && len(trimmed) > 3 {
						listLines = append(listLines, trimmed)
					}
				}
				assert.True(t, len(listLines) >= 1, "Should find at least one list item")
			}

			// Document the current behavior for analysis
			t.Logf("Input: %q", tc.input)
			t.Logf("Output: %q", result)
			t.Logf("Output ends with newline: %v", strings.HasSuffix(result, "\n"))
			t.Logf("Output ends with space: %v", strings.HasSuffix(result, " "))
			t.Logf("Output length: %d", len(result))
		})
	}
}