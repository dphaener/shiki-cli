package diff

import (
	"strings"
	"testing"
)

func TestGenerator_GenerateUnifiedDiff(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name        string
		oldContent  string
		newContent  string
		filePath    string
		expectError bool
		checkResult func(*testing.T, *FileDiff)
	}{
		{
			name:       "simple addition",
			oldContent: "line 1\nline 2",
			newContent: "line 1\nline 2\nline 3",
			filePath:   "test.txt",
			checkResult: func(t *testing.T, diff *FileDiff) {
				if diff.IsBinary {
					t.Error("Expected text diff, got binary")
				}
				if !strings.Contains(diff.UnifiedDiff, "+line 3") {
					t.Error("Expected diff to contain added line")
				}
			},
		},
		{
			name:       "simple deletion",
			oldContent: "line 1\nline 2\nline 3",
			newContent: "line 1\nline 3",
			filePath:   "test.txt",
			checkResult: func(t *testing.T, diff *FileDiff) {
				if diff.IsBinary {
					t.Error("Expected text diff, got binary")
				}
				if !strings.Contains(diff.UnifiedDiff, "-line 2") {
					t.Error("Expected diff to contain deleted line")
				}
			},
		},
		{
			name:       "modification",
			oldContent: "line 1\noriginal line\nline 3",
			newContent: "line 1\nmodified line\nline 3",
			filePath:   "test.txt",
			checkResult: func(t *testing.T, diff *FileDiff) {
				if diff.IsBinary {
					t.Error("Expected text diff, got binary")
				}
				if !strings.Contains(diff.UnifiedDiff, "-original line") {
					t.Error("Expected diff to contain removed line")
				}
				if !strings.Contains(diff.UnifiedDiff, "+modified line") {
					t.Error("Expected diff to contain added line")
				}
			},
		},
		{
			name:       "binary content",
			oldContent: "binary content with null byte: \x00",
			newContent: "modified binary content with null byte: \x00",
			filePath:   "binary.dat",
			checkResult: func(t *testing.T, diff *FileDiff) {
				if !diff.IsBinary {
					t.Error("Expected binary diff, got text")
				}
				if !strings.Contains(diff.UnifiedDiff, "Binary file binary.dat modified") {
					t.Error("Expected binary file modification message")
				}
			},
		},
		{
			name:       "empty files",
			oldContent: "",
			newContent: "",
			filePath:   "empty.txt",
			checkResult: func(t *testing.T, diff *FileDiff) {
				if diff.IsBinary {
					t.Error("Expected text diff, got binary")
				}
				if diff.UnifiedDiff != "" {
					t.Error("Expected empty diff for identical empty files")
				}
			},
		},
		{
			name:       "new file creation",
			oldContent: "",
			newContent: "new file content\nwith multiple lines",
			filePath:   "newfile.txt",
			checkResult: func(t *testing.T, diff *FileDiff) {
				if diff.IsBinary {
					t.Error("Expected text diff, got binary")
				}
				if !strings.Contains(diff.UnifiedDiff, "+new file content") {
					t.Error("Expected diff to show file creation")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diff, err := generator.GenerateUnifiedDiff(test.oldContent, test.newContent, test.filePath)

			if test.expectError && err == nil {
				t.Error("Expected error, but got none")
				return
			}

			if !test.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if test.checkResult != nil {
				test.checkResult(t, diff)
			}

			// Verify basic fields are set
			if diff.FilePath != test.filePath {
				t.Errorf("Expected FilePath %s, got %s", test.filePath, diff.FilePath)
			}
			if diff.OldContent != test.oldContent {
				t.Error("OldContent not preserved")
			}
			if diff.NewContent != test.newContent {
				t.Error("NewContent not preserved")
			}
		})
	}
}

func TestGenerator_IsBinaryContent(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "text content",
			content:  []byte("This is normal text content\nwith newlines"),
			expected: false,
		},
		{
			name:     "content with null bytes",
			content:  []byte("Content with null\x00byte"),
			expected: true,
		},
		{
			name:     "empty content",
			content:  []byte(""),
			expected: false,
		},
		{
			name:     "content with many control characters",
			content:  []byte("\x01\x02\x03\x04\x05\x06\x07\x08"),
			expected: true,
		},
		{
			name:     "valid UTF-8 with normal whitespace",
			content:  []byte("Normal text\twith\ttabs\nand\nnewlines\r\n"),
			expected: false,
		},
		{
			name:     "unicode content",
			content:  []byte("Text with unicode: 你好世界 🌍"),
			expected: false,
		},
		{
			name:     "invalid UTF-8",
			content:  []byte{0xff, 0xfe, 0xfd},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := generator.IsBinaryContent(test.content)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for content: %v", test.expected, result, test.content)
			}
		})
	}
}

func TestGenerator_TruncateDiffIfNeeded(t *testing.T) {
	generator := NewGenerator()
	generator.SetMaxLines(10) // Set small limit for testing

	// Create a large diff (more than 10 lines)
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line " + string(rune('A'+i))
	}
	largeDiff := strings.Join(lines, "\n")

	truncated := generator.truncateDiffIfNeeded(largeDiff)

	// Should contain omission indicator
	if !strings.Contains(truncated, "lines omitted") {
		t.Error("Expected truncated diff to contain omission indicator")
	}

	// Should have fewer lines than original
	originalLines := strings.Split(largeDiff, "\n")
	truncatedLines := strings.Split(truncated, "\n")

	if len(truncatedLines) >= len(originalLines) {
		t.Error("Expected truncated diff to be shorter than original")
	}
}

func TestGenerator_FormatBinaryFileDiff(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name       string
		oldContent string
		newContent string
		filePath   string
		expected   string
	}{
		{
			name:       "file creation",
			oldContent: "",
			newContent: "binary data",
			filePath:   "path/to/new.bin",
			expected:   "Binary file new.bin created (11 bytes)",
		},
		{
			name:       "file deletion",
			oldContent: "binary data",
			newContent: "",
			filePath:   "path/to/old.bin",
			expected:   "Binary file old.bin deleted (11 bytes)",
		},
		{
			name:       "file modification",
			oldContent: "old data",
			newContent: "new binary data",
			filePath:   "path/to/file.bin",
			expected:   "Binary file file.bin modified (8 → 15 bytes)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := generator.formatBinaryFileDiff(test.oldContent, test.newContent, test.filePath)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestSanitizeFilePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal file path",
			input:    "src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "sensitive path with secrets",
			input:    "/home/user/secrets/config.yml",
			expected: "config.yml",
		},
		{
			name:     "env file",
			input:    "project/.env",
			expected: ".env",
		},
		{
			name:     "ssh key",
			input:    "/home/user/.ssh/id_rsa",
			expected: "id_rsa",
		},
		{
			name:     "long relative path",
			input:    strings.Repeat("very/", 20) + "long/path/file.txt",
			expected: "file.txt",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SanitizeFilePath(test.input)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestValidateFileContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "normal content",
			content:  "function main() { console.log('hello'); }",
			expected: true,
		},
		{
			name:     "content with password",
			content:  "config:\n  password=secret123\n  host=localhost",
			expected: false,
		},
		{
			name:     "content with api key",
			content:  "API_KEY=abc123def456",
			expected: false,
		},
		{
			name:     "content with private key",
			content:  "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkq...",
			expected: false,
		},
		{
			name:     "safe configuration",
			content:  "host=localhost\nport=3000\ndatabase=myapp",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ValidateFileContent(test.content)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for content with sensitive patterns", test.expected, result)
			}
		})
	}
}

func TestGenerator_SetMaxLines(t *testing.T) {
	generator := NewGenerator()
	originalMaxLines := generator.maxLines

	// Test setting valid value
	generator.SetMaxLines(50)
	if generator.maxLines != 50 {
		t.Error("Failed to set max lines")
	}

	// Test setting invalid value (should not change)
	generator.SetMaxLines(-1)
	if generator.maxLines != 50 {
		t.Error("Should not accept negative max lines")
	}

	// Reset
	generator.SetMaxLines(originalMaxLines)
}

func TestGenerator_SetContextLines(t *testing.T) {
	generator := NewGenerator()
	originalContextLines := generator.contextLines

	// Test setting valid value
	generator.SetContextLines(5)
	if generator.contextLines != 5 {
		t.Error("Failed to set context lines")
	}

	// Test setting zero (should be valid)
	generator.SetContextLines(0)
	if generator.contextLines != 0 {
		t.Error("Should accept zero context lines")
	}

	// Test setting negative value (should not change from 0)
	generator.SetContextLines(-1)
	if generator.contextLines != 0 {
		t.Error("Should not accept negative context lines")
	}

	// Reset
	generator.SetContextLines(originalContextLines)
}

func TestGetFileSize(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int64
	}{
		{
			name:     "empty string",
			content:  "",
			expected: 0,
		},
		{
			name:     "simple string",
			content:  "hello",
			expected: 5,
		},
		{
			name:     "unicode string",
			content:  "hello 世界",
			expected: 12, // "hello " is 6 bytes, "世界" is 6 bytes (3 bytes each)
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetFileSize(test.content)
			if result != test.expected {
				t.Errorf("Expected %d, got %d", test.expected, result)
			}
		})
	}
}