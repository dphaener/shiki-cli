package diff

import (
	"encoding/json"
	"testing"
)

func TestFileDiff_JSONSerialization(t *testing.T) {
	// Create a test FileDiff
	original := &FileDiff{
		FilePath:    "test/file.txt",
		OldContent:  "old content\nline 2",
		NewContent:  "new content\nline 2",
		UnifiedDiff: "@@ -1,2 +1,2 @@\n-old content\n+new content\n line 2",
		IsBinary:    false,
		IsLarge:     false,
		TotalLines:  4,
	}

	// Test marshaling
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal FileDiff: %v", err)
	}

	// Test unmarshaling
	var restored FileDiff
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal FileDiff: %v", err)
	}

	// Verify all fields are preserved
	if restored.FilePath != original.FilePath {
		t.Errorf("FilePath mismatch: got %s, want %s", restored.FilePath, original.FilePath)
	}
	if restored.OldContent != original.OldContent {
		t.Error("OldContent mismatch")
	}
	if restored.NewContent != original.NewContent {
		t.Error("NewContent mismatch")
	}
	if restored.UnifiedDiff != original.UnifiedDiff {
		t.Error("UnifiedDiff mismatch")
	}
	if restored.IsBinary != original.IsBinary {
		t.Errorf("IsBinary mismatch: got %t, want %t", restored.IsBinary, original.IsBinary)
	}
	if restored.IsLarge != original.IsLarge {
		t.Errorf("IsLarge mismatch: got %t, want %t", restored.IsLarge, original.IsLarge)
	}
	if restored.TotalLines != original.TotalLines {
		t.Errorf("TotalLines mismatch: got %d, want %d", restored.TotalLines, original.TotalLines)
	}
}

func TestFileDiff_JSONOmitEmpty(t *testing.T) {
	// Test with nil diff (should be omitted in ToolResultPart)
	var nilDiff *FileDiff = nil

	// Test serializing a struct containing a nil FileDiff pointer
	testStruct := struct {
		ID   string    `json:"id"`
		Diff *FileDiff `json:"diff,omitempty"`
	}{
		ID:   "test",
		Diff: nilDiff,
	}

	data, err := json.Marshal(testStruct)
	if err != nil {
		t.Fatalf("Failed to marshal struct with nil diff: %v", err)
	}

	// Verify that the diff field is omitted
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := result["diff"]; exists {
		t.Error("Expected diff field to be omitted when nil")
	}

	if result["id"] != "test" {
		t.Error("Expected id field to be present")
	}
}