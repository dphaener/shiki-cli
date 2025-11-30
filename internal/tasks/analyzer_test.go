package tasks

import (
	"strings"
	"testing"
)

func createTestAnalyzer() *ResponseAnalyzer {
	taskStructure := createTestTaskStructure()
	// Add the task and workpackage maps that were missing
	taskStructure.TaskMap = map[string]*Task{
		"T001": &taskStructure.WorkPackages[0].Tasks[0],
		"T002": &taskStructure.WorkPackages[0].Tasks[1],
		"T003": &taskStructure.WorkPackages[0].Tasks[2],
		"T004": &taskStructure.WorkPackages[1].Tasks[0],
		"T005": &taskStructure.WorkPackages[1].Tasks[1],
	}
	taskStructure.WPMap = map[string]*WorkPackage{
		"WP01": &taskStructure.WorkPackages[0],
		"WP02": &taskStructure.WorkPackages[1],
	}

	return NewResponseAnalyzer(taskStructure)
}

func TestResponseAnalyzer_DetectTaskReferences(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single task reference",
			content:  "I am working on T001 to initialize the project.",
			expected: []string{"T001"},
		},
		{
			name:     "multiple task references",
			content:  "Completed T001 and T002, now starting T003.",
			expected: []string{"T001", "T002", "T003"},
		},
		{
			name:     "duplicate task references",
			content:  "T001 is done. T001 was successful.",
			expected: []string{"T001"},
		},
		{
			name:     "no task references",
			content:  "This is just regular text without any task IDs.",
			expected: []string{},
		},
		{
			name:     "invalid task references",
			content:  "T999 does not exist in our task structure.",
			expected: []string{},
		},
		{
			name:     "mixed valid and invalid",
			content:  "T001 is valid but T999 is not valid.",
			expected: []string{"T001"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.DetectTaskReferences(tc.content)

			if len(result) != len(tc.expected) {
				t.Errorf("expected %d task references, got %d", len(tc.expected), len(result))
				return
			}

			// Convert to map for easier comparison
			expectedMap := make(map[string]bool)
			for _, taskID := range tc.expected {
				expectedMap[taskID] = true
			}

			for _, taskID := range result {
				if !expectedMap[taskID] {
					t.Errorf("unexpected task ID: %s", taskID)
				}
			}
		})
	}
}

func TestResponseAnalyzer_AnalyzeResponse_ExplicitPatterns(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name           string
		content        string
		expectedTaskID string
		expectedStatus TaskProgressState
		minConfidence  float64
	}{
		{
			name:           "task completed explicitly",
			content:        "Completed T001 successfully.",
			expectedTaskID: "T001",
			expectedStatus: TaskStateCompleted,
			minConfidence:  0.8,
		},
		{
			name:           "task started explicitly",
			content:        "Starting T002 now.",
			expectedTaskID: "T002",
			expectedStatus: TaskStateInProgress,
			minConfidence:  0.8,
		},
		{
			name:           "task blocked explicitly",
			content:        "T003 blocked: missing API credentials",
			expectedTaskID: "T003",
			expectedStatus: TaskStateBlocked,
			minConfidence:  0.8,
		},
		{
			name:           "task skipped explicitly",
			content:        "Skipping T004 as it's no longer needed.",
			expectedTaskID: "T004",
			expectedStatus: TaskStateSkipped,
			minConfidence:  0.8,
		},
		{
			name:           "task with checkmark",
			content:        "✓ T001 Initialize project",
			expectedTaskID: "T001",
			expectedStatus: TaskStateCompleted,
			minConfidence:  0.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updates := analyzer.AnalyzeResponse(tc.content)

			found := false
			for _, update := range updates {
				if update.TaskID == tc.expectedTaskID && update.Status == tc.expectedStatus {
					found = true
					if update.Confidence < tc.minConfidence {
						t.Errorf("expected confidence >= %.2f, got %.2f", tc.minConfidence, update.Confidence)
					}
					break
				}
			}

			if !found {
				t.Errorf("expected to find update for task %s with status %s", tc.expectedTaskID, tc.expectedStatus)
			}
		})
	}
}

func TestResponseAnalyzer_AnalyzeResponse_ImplicitPatterns(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name           string
		content        string
		expectedTaskID string
		expectedStatus TaskProgressState
		minConfidence  float64
	}{
		{
			name:           "task completion implied",
			content:        "T001 has been successfully implemented and is working correctly.",
			expectedTaskID: "T001",
			expectedStatus: TaskStateCompleted,
			minConfidence:  0.5,
		},
		{
			name:           "task in progress implied",
			content:        "Currently working on T002 to set up the database connection.",
			expectedTaskID: "T002",
			expectedStatus: TaskStateInProgress,
			minConfidence:  0.5,
		},
		{
			name:           "task blocked implied",
			content:        "T003 cannot proceed due to missing API credentials from the team.",
			expectedTaskID: "T003",
			expectedStatus: TaskStateBlocked,
			minConfidence:  0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updates := analyzer.AnalyzeResponse(tc.content)

			found := false
			for _, update := range updates {
				if update.TaskID == tc.expectedTaskID && update.Status == tc.expectedStatus {
					found = true
					if update.Confidence < tc.minConfidence {
						t.Errorf("expected confidence >= %.2f, got %.2f", tc.minConfidence, update.Confidence)
					}
					break
				}
			}

			if !found {
				t.Errorf("expected to find update for task %s with status %s", tc.expectedTaskID, tc.expectedStatus)
			}
		})
	}
}

func TestResponseAnalyzer_InferTaskStatus(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name     string
		content  string
		taskID   string
		expected TaskProgressState
	}{
		{
			name:     "clear completion",
			content:  "T001 completed successfully",
			taskID:   "T001",
			expected: TaskStateCompleted,
		},
		{
			name:     "clear in progress",
			content:  "Currently implementing T002",
			taskID:   "T002",
			expected: TaskStateInProgress,
		},
		{
			name:     "no clear status",
			content:  "T003 is mentioned without status",
			taskID:   "T003",
			expected: TaskStatePending,
		},
		{
			name:     "invalid task",
			content:  "T999 completed",
			taskID:   "T999",
			expected: TaskStatePending,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.InferTaskStatus(tc.content, tc.taskID)
			if result != tc.expected {
				t.Errorf("expected status %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestResponseAnalyzer_AddCustomKeywords(t *testing.T) {
	analyzer := createTestAnalyzer()

	// Add custom keywords
	customKeywords := []string{"initialized", "bootstrapped"}
	analyzer.AddCustomKeywords(TaskStateCompleted, customKeywords)

	keywords := analyzer.GetKeywords()
	completedKeywords := keywords[TaskStateCompleted]

	// Check if custom keywords were added
	foundInitialized := false
	foundBootstrapped := false

	for _, keyword := range completedKeywords {
		if keyword == "initialized" {
			foundInitialized = true
		}
		if keyword == "bootstrapped" {
			foundBootstrapped = true
		}
	}

	if !foundInitialized {
		t.Error("expected 'initialized' keyword to be added")
	}
	if !foundBootstrapped {
		t.Error("expected 'bootstrapped' keyword to be added")
	}

	// Test that the custom keyword works
	content := "T001 has been initialized and is ready"
	updates := analyzer.AnalyzeResponse(content)

	found := false
	for _, update := range updates {
		if update.TaskID == "T001" && update.Status == TaskStateCompleted {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to detect completion using custom keyword")
	}
}

func TestResponseAnalyzer_ConfidenceCalculation(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name        string
		content     string
		minConfidence float64
		maxConfidence float64
	}{
		{
			name:          "high confidence explicit",
			content:       "Completed T001 successfully",
			minConfidence: 0.8,
			maxConfidence: 1.0,
		},
		{
			name:          "medium confidence implicit",
			content:       "T001 is done and working",
			minConfidence: 0.5,
			maxConfidence: 0.8,
		},
		{
			name:          "boosted confidence with multiple indicators",
			content:       "T001 completed successfully ✓ finished",
			minConfidence: 0.8,
			maxConfidence: 1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updates := analyzer.AnalyzeResponse(tc.content)

			if len(updates) == 0 {
				t.Fatal("expected at least one update")
			}

			update := updates[0]
			if update.Confidence < tc.minConfidence || update.Confidence > tc.maxConfidence {
				t.Errorf("expected confidence between %.2f and %.2f, got %.2f",
					tc.minConfidence, tc.maxConfidence, update.Confidence)
			}
		})
	}
}

func TestResponseAnalyzer_ReasonExtraction(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name           string
		content        string
		expectedReason string
	}{
		{
			name:           "blocked with reason",
			content:        "T001 blocked: missing API credentials from the backend team",
			expectedReason: "missing API credentials from the backend team",
		},
		{
			name:           "skipped with reason",
			content:        "Skipping T002 because it's no longer needed for this iteration",
			expectedReason: "it's no longer needed for this iteration",
		},
		{
			name:           "error with reason",
			content:        "T003 failed due to network connectivity issues",
			expectedReason: "network connectivity issues",
		},
		{
			name:           "no reason provided",
			content:        "T004 is blocked",
			expectedReason: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updates := analyzer.AnalyzeResponse(tc.content)

			if len(updates) == 0 {
				t.Fatal("expected at least one update")
			}

			update := updates[0]
			if tc.expectedReason == "" {
				// Allow empty or short reasons
				if len(update.Reason) > 50 {
					t.Errorf("expected short or empty reason, got: %s", update.Reason)
				}
			} else {
				if update.Reason == "" {
					t.Error("expected reason to be extracted")
				} else if !containsSubstring(update.Reason, tc.expectedReason) {
					t.Errorf("expected reason to contain '%s', got: %s", tc.expectedReason, update.Reason)
				}
			}
		})
	}
}

func TestResponseAnalyzer_DuplicateHandling(t *testing.T) {
	analyzer := createTestAnalyzer()

	content := "Starting T001. T001 is in progress. Working on T001."
	updates := analyzer.AnalyzeResponse(content)

	// Should only get one update for T001 despite multiple mentions
	t001Updates := 0
	for _, update := range updates {
		if update.TaskID == "T001" {
			t001Updates++
		}
	}

	if t001Updates != 1 {
		t.Errorf("expected exactly 1 update for T001, got %d", t001Updates)
	}
}

func TestResponseAnalyzer_NoFalsePositives(t *testing.T) {
	analyzer := createTestAnalyzer()

	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "task mentioned in unrelated context",
			content: "The T001 specification document needs to be reviewed.",
		},
		{
			name:    "task mentioned in question",
			content: "Should we start T002 tomorrow?",
		},
		{
			name:    "task mentioned in planning",
			content: "T003 will require API credentials once we get them.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updates := analyzer.AnalyzeResponse(tc.content)

			// Should have low confidence or no updates for ambiguous mentions
			for _, update := range updates {
				if update.Confidence > 0.7 {
					t.Errorf("expected low confidence for ambiguous mention, got %.2f", update.Confidence)
				}
			}
		})
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr))))
}