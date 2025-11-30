package tasks

import (
	"testing"
)

func TestTasksParser_ParseContent(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected int // expected number of work packages
		wantErr  bool
	}{
		{
			name: "valid tasks file",
			content: `# Implementation Tasks

## Work Packages

### WP01: Core Infrastructure
**Priority**: P0
**Goal**: Create the foundational components
**Dependencies**: None

#### Subtasks
- [ ] T001: Initialize project structure [P]
- [ ] T002: Create basic types
- [ ] T003: Implement core logic

### WP02: Integration Layer
**Priority**: P1
**Goal**: Integrate with existing systems
**Dependencies**: WP01

#### Subtasks
- [ ] T004: Setup event handlers
- [ ] T005: Connect to database
`,
			expected: 2,
			wantErr:  false,
		},
		{
			name: "empty content",
			content: "",
			expected: 0,
			wantErr:  true,
		},
		{
			name: "no work packages",
			content: `# Implementation Tasks

This is just some text without work packages.
`,
			expected: 0,
			wantErr:  true,
		},
	}

	parser := NewTasksParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.ParseContent(tc.content)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result.WorkPackages) != tc.expected {
				t.Errorf("expected %d work packages, got %d", tc.expected, len(result.WorkPackages))
			}
		})
	}
}

func TestTasksParser_ExtractWorkPackages(t *testing.T) {
	content := `# Implementation Tasks

### WP01: Core Infrastructure
**Priority**: P0
**Goal**: Create the foundational components
**Dependencies**: None

#### Subtasks
- [ ] T001: Initialize project structure [P]
- [ ] T002: Create basic types
- [ ] T003: Implement core logic

### WP02: Integration Layer
**Priority**: P1
**Goal**: Integrate with existing systems
**Dependencies**: WP01

#### Subtasks
- [ ] T004: Setup event handlers
- [ ] T005: Connect to database
`

	parser := NewTasksParser()
	workPackages, err := parser.extractWorkPackages(content)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workPackages) != 2 {
		t.Fatalf("expected 2 work packages, got %d", len(workPackages))
	}

	// Test first work package
	wp1 := workPackages[0]
	if wp1.ID != "WP01" {
		t.Errorf("expected ID 'WP01', got '%s'", wp1.ID)
	}
	if wp1.Name != "Core Infrastructure" {
		t.Errorf("expected name 'Core Infrastructure', got '%s'", wp1.Name)
	}
	if wp1.Priority != "P0" {
		t.Errorf("expected priority 'P0', got '%s'", wp1.Priority)
	}
	if wp1.Goal != "Create the foundational components" {
		t.Errorf("expected goal 'Create the foundational components', got '%s'", wp1.Goal)
	}
	if len(wp1.Dependencies) != 0 {
		t.Errorf("expected no dependencies, got %d", len(wp1.Dependencies))
	}
	if len(wp1.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(wp1.Tasks))
	}

	// Test task details
	task1 := wp1.Tasks[0]
	if task1.ID != "T001" {
		t.Errorf("expected task ID 'T001', got '%s'", task1.ID)
	}
	if task1.Description != "Initialize project structure" {
		t.Errorf("expected description 'Initialize project structure', got '%s'", task1.Description)
	}
	if task1.Priority != "P" {
		t.Errorf("expected priority 'P', got '%s'", task1.Priority)
	}

	// Test second work package
	wp2 := workPackages[1]
	if wp2.ID != "WP02" {
		t.Errorf("expected ID 'WP02', got '%s'", wp2.ID)
	}
	if len(wp2.Dependencies) != 1 || wp2.Dependencies[0] != "WP01" {
		t.Errorf("expected dependencies ['WP01'], got %v", wp2.Dependencies)
	}
}

func TestTasksParser_ParseTask(t *testing.T) {
	parser := NewTasksParser()

	testCases := []struct {
		name           string
		line           string
		expectedID     string
		expectedDesc   string
		expectedPrio   string
		shouldBeNil    bool
	}{
		{
			name:         "valid task with priority",
			line:         "- [ ] T001: Initialize project structure [P]",
			expectedID:   "T001",
			expectedDesc: "Initialize project structure",
			expectedPrio: "P",
			shouldBeNil:  false,
		},
		{
			name:         "valid task without priority",
			line:         "- [ ] T002: Create basic types",
			expectedID:   "T002",
			expectedDesc: "Create basic types",
			expectedPrio: "",
			shouldBeNil:  false,
		},
		{
			name:        "not a task line",
			line:        "This is just regular text",
			shouldBeNil: true,
		},
		{
			name:        "empty line",
			line:        "",
			shouldBeNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task, err := parser.parseTask(tc.line, 1)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.shouldBeNil {
				if task != nil {
					t.Errorf("expected nil task, got %+v", task)
				}
				return
			}

			if task == nil {
				t.Errorf("expected task, got nil")
				return
			}

			if task.ID != tc.expectedID {
				t.Errorf("expected ID '%s', got '%s'", tc.expectedID, task.ID)
			}
			if task.Description != tc.expectedDesc {
				t.Errorf("expected description '%s', got '%s'", tc.expectedDesc, task.Description)
			}
			if task.Priority != tc.expectedPrio {
				t.Errorf("expected priority '%s', got '%s'", tc.expectedPrio, task.Priority)
			}
		})
	}
}

func TestParseDependencies(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single dependency",
			input:    "WP01",
			expected: []string{"WP01"},
		},
		{
			name:     "multiple dependencies",
			input:    "WP01, WP02, WP03",
			expected: []string{"WP01", "WP02", "WP03"},
		},
		{
			name:     "dependencies with extra spaces",
			input:    " WP01 , WP02 , WP03 ",
			expected: []string{"WP01", "WP02", "WP03"},
		},
		{
			name:     "none dependency",
			input:    "None",
			expected: []string{},
		},
		{
			name:     "empty dependency",
			input:    "",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseDependencies(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("expected %d dependencies, got %d", len(tc.expected), len(result))
				return
			}

			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("expected dependency[%d] to be '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestTaskStructure_LookupMaps(t *testing.T) {
	content := `# Implementation Tasks

### WP01: Core Infrastructure
**Priority**: P0
**Goal**: Create the foundational components
**Dependencies**: None

#### Subtasks
- [ ] T001: Initialize project structure
- [ ] T002: Create basic types
`

	parser := NewTasksParser()
	taskStructure, err := parser.ParseContent(content)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test task lookup
	task, exists := taskStructure.TaskMap["T001"]
	if !exists {
		t.Errorf("expected to find task T001 in TaskMap")
	}
	if task.Description != "Initialize project structure" {
		t.Errorf("expected description 'Initialize project structure', got '%s'", task.Description)
	}

	// Test work package lookup
	wp, exists := taskStructure.WPMap["WP01"]
	if !exists {
		t.Errorf("expected to find work package WP01 in WPMap")
	}
	if wp.Name != "Core Infrastructure" {
		t.Errorf("expected name 'Core Infrastructure', got '%s'", wp.Name)
	}

	// Test non-existent lookups
	_, exists = taskStructure.TaskMap["T999"]
	if exists {
		t.Errorf("should not find non-existent task T999")
	}

	_, exists = taskStructure.WPMap["WP999"]
	if exists {
		t.Errorf("should not find non-existent work package WP999")
	}
}

func TestTasksParser_MalformedContent(t *testing.T) {
	parser := NewTasksParser()

	testCases := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "work package without tasks section",
			content: `### WP01: Test Package
**Priority**: P0
**Goal**: Test goal
**Dependencies**: None
`,
			wantErr: false, // Should handle gracefully
		},
		{
			name: "tasks without work package header",
			content: `#### Subtasks
- [ ] T001: Orphaned task
`,
			wantErr: true, // No work packages found
		},
		{
			name: "malformed task lines",
			content: `### WP01: Test Package
#### Subtasks
- T001: Missing checkbox
- [ ] Missing task ID: description
`,
			wantErr: false, // Should skip malformed tasks
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.ParseContent(tc.content)

			if tc.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}