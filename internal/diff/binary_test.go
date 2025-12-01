package diff

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TestBinaryFileDetection tests detection of various binary file formats
func TestBinaryFileDetection(t *testing.T) {
	generator := NewGenerator()

	testCases := []struct {
		name        string
		oldContent  string
		newContent  string
		fileName    string
		expectBinary bool
		description string
	}{
		{
			name:        "jpeg_file",
			oldContent:  "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00",
			newContent:  "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x02\x00H\x00H\x00\x00",
			fileName:    "image.jpg",
			expectBinary: true,
			description: "JPEG image with modified metadata",
		},
		{
			name:        "png_file",
			oldContent:  "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10",
			newContent:  "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x20",
			fileName:    "image.png",
			expectBinary: true,
			description: "PNG image with modified dimensions",
		},
		{
			name:        "pdf_file",
			oldContent:  "%PDF-1.4\n%\xe2\xe3\xcf\xd3\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\n1 0 obj\n<<\n/Type /Catalog",
			newContent:  "%PDF-1.5\n%\xe2\xe3\xcf\xd3\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\n1 0 obj\n<<\n/Type /Catalog",
			fileName:    "document.pdf",
			expectBinary: true,
			description: "PDF document with version change",
		},
		{
			name:        "zip_file",
			oldContent:  "PK\x03\x04\x14\x00\x00\x00\x08\x00",
			newContent:  "PK\x03\x04\x14\x00\x00\x00\x08\x01",
			fileName:    "archive.zip",
			expectBinary: true,
			description: "ZIP archive with modified header",
		},
		{
			name:        "executable_file",
			oldContent:  "\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00",
			newContent:  "\x7fELF\x02\x01\x01\x00\x00\x00\x00\x01\x00\x00\x00\x00",
			fileName:    "binary",
			expectBinary: true,
			description: "ELF executable",
		},
		{
			name:        "null_byte_text",
			oldContent:  "hello\x00world",
			newContent:  "hello\x00universe",
			fileName:    "mixed.txt",
			expectBinary: true,
			description: "Text file with embedded null bytes",
		},
		{
			name:        "pure_text",
			oldContent:  "Hello world\nThis is a text file",
			newContent:  "Hello universe\nThis is a text file",
			fileName:    "text.txt",
			expectBinary: false,
			description: "Pure text content",
		},
		{
			name:        "utf8_text",
			oldContent:  "Hello 世界\nUnicode text",
			newContent:  "Hello 宇宙\nUnicode text",
			fileName:    "unicode.txt",
			expectBinary: false,
			description: "UTF-8 text with Unicode characters",
		},
		{
			name:        "json_file",
			oldContent:  `{"name": "test", "value": 123}`,
			newContent:  `{"name": "test", "value": 456}`,
			fileName:    "data.json",
			expectBinary: false,
			description: "JSON file with numeric change",
		},
		{
			name:        "high_ascii_text",
			oldContent:  "Café naïve résumé",
			newContent:  "Café naïve curriculum vitae",
			fileName:    "accents.txt",
			expectBinary: false,
			description: "Text with high ASCII characters",
		},
		{
			name:        "mixed_content_mostly_binary",
			oldContent:  "\x00\x01\x02\x03hello\x04\x05\x06\x07",
			newContent:  "\x00\x01\x02\x03goodbye\x04\x05\x06\x07",
			fileName:    "mixed.dat",
			expectBinary: true,
			description: "Mixed content with significant binary data",
		},
		{
			name:        "control_characters",
			oldContent:  "Line 1\x1b[32mGreen text\x1b[0m\nLine 2",
			newContent:  "Line 1\x1b[31mRed text\x1b[0m\nLine 2",
			fileName:    "colored.txt",
			expectBinary: false,
			description: "Text with ANSI control sequences",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileDiff, err := generator.GenerateUnifiedDiff(tc.oldContent, tc.newContent, tc.fileName)
			if err != nil {
				t.Fatalf("Failed to generate diff for %s: %v", tc.description, err)
			}

			if fileDiff.IsBinary != tc.expectBinary {
				t.Errorf("Binary detection failed for %s: expected %v, got %v",
					tc.description, tc.expectBinary, fileDiff.IsBinary)
			}

			// For binary files, ensure we get a descriptive message
			if tc.expectBinary {
				if !strings.Contains(fileDiff.UnifiedDiff, "Binary file") {
					t.Errorf("Binary file diff should contain 'Binary file' message, got: %s",
						fileDiff.UnifiedDiff)
				}
				if !strings.Contains(fileDiff.UnifiedDiff, tc.fileName) {
					t.Errorf("Binary file diff should contain filename, got: %s",
						fileDiff.UnifiedDiff)
				}
			} else {
				// For text files, ensure we get actual diff content
				if fileDiff.UnifiedDiff == "" && tc.oldContent != tc.newContent {
					t.Error("Text file with changes should produce non-empty diff")
				}
			}

			// Verify file path is preserved
			if fileDiff.FilePath != tc.fileName {
				t.Errorf("File path not preserved: expected %s, got %s",
					tc.fileName, fileDiff.FilePath)
			}
		})
	}
}

// TestLargeFileHandling tests diff generation with very large files
func TestLargeFileHandling(t *testing.T) {
	generator := NewGenerator()

	// Test with different file sizes
	sizes := []struct {
		name      string
		lines     int
		expectLarge bool
	}{
		{"small_file", 100, false},
		{"medium_file", 500, false},
		{"large_file", 2000, true},
		{"very_large_file", 5000, true},
	}

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			// Generate large content
			oldContent := generateLargeContent(size.lines, "original")
			newContent := generateLargeContent(size.lines, "modified")

			fileDiff, err := generator.GenerateUnifiedDiff(oldContent, newContent, "large.txt")
			if err != nil {
				t.Fatalf("Failed to generate diff for %d lines: %v", size.lines, err)
			}

			// Check if large file is properly detected
			if fileDiff.IsLarge != size.expectLarge {
				t.Errorf("Large file detection failed for %d lines: expected %v, got %v",
					size.lines, size.expectLarge, fileDiff.IsLarge)
			}

			// Ensure diff generation completes in reasonable time
			// This is implicit - if we get here, it completed

			// Check that total lines are tracked
			if fileDiff.TotalLines == 0 && size.lines > 0 {
				t.Error("TotalLines should be greater than 0 for non-empty files")
			}

			// For large files, check that truncation is applied
			if size.expectLarge {
				diffLines := strings.Split(fileDiff.UnifiedDiff, "\n")
				if len(diffLines) > 1000 { // Should be limited by maxLines
					t.Errorf("Large file diff not properly truncated: %d lines", len(diffLines))
				}
			}
		})
	}
}

// TestBinaryFileSize tests size calculation for binary files
func TestBinaryFileSize(t *testing.T) {
	generator := NewGenerator()

	testCases := []struct {
		name       string
		oldContent string
		newContent string
		fileName   string
	}{
		{
			name:       "size_increase",
			oldContent: "\x00\x01\x02\x03",
			newContent: "\x00\x01\x02\x03\x04\x05",
			fileName:   "binary.dat",
		},
		{
			name:       "size_decrease",
			oldContent: "\xFF\xFE\xFD\xFC\xFB\xFA",
			newContent: "\xFF\xFE\xFD",
			fileName:   "shrunk.bin",
		},
		{
			name:       "same_size",
			oldContent: "\x10\x20\x30\x40",
			newContent: "\x50\x60\x70\x80",
			fileName:   "modified.bin",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileDiff, err := generator.GenerateUnifiedDiff(tc.oldContent, tc.newContent, tc.fileName)
			if err != nil {
				t.Fatalf("Failed to generate binary diff: %v", err)
			}

			if !fileDiff.IsBinary {
				t.Fatal("File should be detected as binary")
			}

			// Check that the diff message contains size information
			oldSize := len(tc.oldContent)
			newSize := len(tc.newContent)

			expectedSubstrings := []string{
				"Binary file",
				tc.fileName,
				fmt.Sprintf("%d", oldSize),
				fmt.Sprintf("%d", newSize),
				"bytes",
			}

			for _, substring := range expectedSubstrings {
				if !strings.Contains(fileDiff.UnifiedDiff, substring) {
					t.Errorf("Binary diff should contain '%s', got: %s",
						substring, fileDiff.UnifiedDiff)
				}
			}
		})
	}
}

// TestEdgeCaseBinaryDetection tests edge cases for binary detection
func TestEdgeCaseBinaryDetection(t *testing.T) {
	generator := NewGenerator()

	testCases := []struct {
		name        string
		content     string
		expectBinary bool
		description string
	}{
		{
			name:        "empty_file",
			content:     "",
			expectBinary: false,
			description: "Empty file should be treated as text",
		},
		{
			name:        "single_null",
			content:     "\x00",
			expectBinary: true,
			description: "Single null byte should be binary",
		},
		{
			name:        "null_at_end",
			content:     "hello world\x00",
			expectBinary: true,
			description: "Null byte at end should make it binary",
		},
		{
			name:        "tabs_and_newlines",
			content:     "hello\tworld\n\rtest",
			expectBinary: false,
			description: "Tabs and newlines should be text",
		},
		{
			name:        "high_bytes_valid_utf8",
			content:     "Hello 世界 🌍",
			expectBinary: false,
			description: "Valid UTF-8 with high bytes should be text",
		},
		{
			name:        "invalid_utf8_sequence",
			content:     "Hello \xFF\xFE world",
			expectBinary: true,
			description: "Invalid UTF-8 sequence should be binary",
		},
		{
			name:        "mostly_printable_with_few_nulls",
			content:     strings.Repeat("hello world ", 100) + "\x00\x00",
			expectBinary: true,
			description: "Even mostly text with nulls should be binary",
		},
		{
			name:        "all_control_chars",
			content:     "\x01\x02\x03\x04\x05\x06\x07\x08",
			expectBinary: true,
			description: "All control characters should be binary",
		},
		{
			name:        "mixed_printable_control",
			content:     "ABC\x01\x02\x03DEF",
			expectBinary: true,
			description: "Mixed printable and control chars should be binary",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with same content to focus on binary detection
			fileDiff, err := generator.GenerateUnifiedDiff(tc.content, tc.content, "test.dat")
			if err != nil {
				t.Fatalf("Failed to generate diff for %s: %v", tc.description, err)
			}

			if fileDiff.IsBinary != tc.expectBinary {
				t.Errorf("Binary detection failed for %s: expected %v, got %v",
					tc.description, tc.expectBinary, fileDiff.IsBinary)
			}
		})
	}
}

// generateLargeContent creates content with the specified number of lines
func generateLargeContent(lines int, prefix string) string {
	var content bytes.Buffer

	for i := 0; i < lines; i++ {
		lineNum := i + 1
		switch {
		case i%20 == 0:
			content.WriteString(fmt.Sprintf("// %s: Section header %d\n", prefix, lineNum/20+1))
		case i%10 == 0:
			content.WriteString(fmt.Sprintf("func %s_Function_%d() {\n", prefix, lineNum))
		case i%10 == 9:
			content.WriteString("}\n")
		case i%5 == 0:
			content.WriteString(fmt.Sprintf("    // %s comment line %d\n", prefix, lineNum))
		default:
			content.WriteString(fmt.Sprintf("    %s_variable_%d := \"value_%d\"\n", prefix, lineNum, lineNum))
		}
	}

	return content.String()
}

// TestBinaryFileRendering tests that binary files are rendered appropriately
func TestBinaryFileRendering(t *testing.T) {
	generator := NewGenerator()
	renderer := NewRenderer()

	// Generate a binary file diff
	oldBinary := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10"
	newBinary := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x20"

	fileDiff, err := generator.GenerateUnifiedDiff(oldBinary, newBinary, "image.png")
	if err != nil {
		t.Fatalf("Failed to generate binary diff: %v", err)
	}

	if !fileDiff.IsBinary {
		t.Fatal("File should be detected as binary")
	}

	// Test rendering at different widths
	widths := []int{40, 80, 120}
	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			rendered := renderer.FormatDiffForTerminal(fileDiff, width)

			if rendered == "" {
				t.Error("Binary file diff should produce non-empty rendering")
			}

			// Should contain binary file indicator
			if !strings.Contains(rendered, "Binary file") {
				t.Error("Rendered binary diff should contain 'Binary file' indicator")
			}

			// Should contain filename
			if !strings.Contains(rendered, "image.png") {
				t.Error("Rendered binary diff should contain filename")
			}

			// Should not contain actual binary content
			if strings.Contains(rendered, "\x89PNG") {
				t.Error("Rendered binary diff should not contain raw binary data")
			}
		})
	}
}