package tasks

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TasksParser handles parsing of tasks.md files
type TasksParser struct{}

// NewTasksParser creates a new parser instance
func NewTasksParser() *TasksParser {
	return &TasksParser{}
}

// WorkPackage represents a work package parsed from tasks.md
type WorkPackage struct {
	ID          string   // e.g., "WP01"
	Name        string   // e.g., "Core Progress Tracking Infrastructure"
	Priority    string   // e.g., "P0"
	Goal        string   // Goal description
	Dependencies []string // e.g., ["WP01", "WP02"]
	Tasks       []Task   // List of tasks in this work package
}

// Task represents a single task within a work package
type Task struct {
	ID          string // e.g., "T001"
	Description string // Task description
	Priority    string // Priority marker [P] if present
}

// TaskStructure represents the full structure parsed from tasks.md
type TaskStructure struct {
	WorkPackages []WorkPackage
	TaskMap      map[string]*Task // Maps task ID to task for quick lookup
	WPMap        map[string]*WorkPackage // Maps WP ID to work package
}

// ParseTasksFile parses a tasks.md file and extracts the task structure
func (p *TasksParser) ParseTasksFile(filePath string) (*TaskStructure, error) {
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	return p.ParseContent(content)
}

// ParseContent parses the content of a tasks.md file
func (p *TasksParser) ParseContent(content string) (*TaskStructure, error) {
	workPackages, err := p.extractWorkPackages(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract work packages: %w", err)
	}

	// Build lookup maps
	taskMap := make(map[string]*Task)
	wpMap := make(map[string]*WorkPackage)

	for i := range workPackages {
		wp := &workPackages[i]
		wpMap[wp.ID] = wp

		for j := range wp.Tasks {
			task := &wp.Tasks[j]
			taskMap[task.ID] = task
		}
	}

	return &TaskStructure{
		WorkPackages: workPackages,
		TaskMap:      taskMap,
		WPMap:        wpMap,
	}, nil
}

// extractWorkPackages finds and parses all work packages in the content
func (p *TasksParser) extractWorkPackages(content string) ([]WorkPackage, error) {
	var workPackages []WorkPackage
	lines := strings.Split(content, "\n")

	// Regex patterns for parsing
	wpHeaderRegex := regexp.MustCompile(`^### (WP\d+):\s*(.+)$`)
	priorityRegex := regexp.MustCompile(`^\*\*Priority\*\*:\s*(.+)$`)
	goalRegex := regexp.MustCompile(`^\*\*Goal\*\*:\s*(.+)$`)
	dependenciesRegex := regexp.MustCompile(`^\*\*Dependencies\*\*:\s*(.+)$`)

	var currentWP *WorkPackage
	var inSubtasks bool

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Work package header
		if match := wpHeaderRegex.FindStringSubmatch(line); match != nil {
			// Save previous work package
			if currentWP != nil {
				workPackages = append(workPackages, *currentWP)
			}

			// Start new work package
			currentWP = &WorkPackage{
				ID:   match[1],
				Name: match[2],
			}
			inSubtasks = false
		} else if currentWP != nil {
			// Priority
			if match := priorityRegex.FindStringSubmatch(line); match != nil {
				currentWP.Priority = match[1]
			} else if match := goalRegex.FindStringSubmatch(line); match != nil {
				currentWP.Goal = match[1]
			} else if match := dependenciesRegex.FindStringSubmatch(line); match != nil {
				deps := strings.TrimSpace(match[1])
				if deps != "None" && deps != "" {
					// Parse dependencies (assuming comma-separated)
					currentWP.Dependencies = parseDependencies(deps)
				}
			} else if strings.Contains(line, "#### Subtasks") {
				inSubtasks = true
			} else if inSubtasks {
				// Parse task
				if task, err := p.parseTask(line, i+1); err == nil && task != nil {
					currentWP.Tasks = append(currentWP.Tasks, *task)
				}
			}
		}
	}

	// Save the last work package
	if currentWP != nil {
		workPackages = append(workPackages, *currentWP)
	}

	if len(workPackages) == 0 {
		return nil, fmt.Errorf("no work packages found in content")
	}

	return workPackages, nil
}

// parseTask parses a single task line
func (p *TasksParser) parseTask(line string, lineNum int) (*Task, error) {
	line = strings.TrimSpace(line)

	// Match task pattern: - [ ] T001: Description [P]
	taskRegex := regexp.MustCompile(`^-\s*\[\s*\]\s*(T\d+):\s*([^[]+)(?:\s*\[P\])?`)
	match := taskRegex.FindStringSubmatch(line)

	if match == nil {
		// Not a task line
		return nil, nil
	}

	task := &Task{
		ID:          match[1],
		Description: strings.TrimSpace(match[2]),
	}

	// Check for priority marker
	if strings.Contains(line, "[P]") {
		task.Priority = "P"
	}

	return task, nil
}

// parseDependencies parses dependency strings like "WP01, WP02" or "WP01"
func parseDependencies(deps string) []string {
	var dependencies []string
	parts := strings.Split(deps, ",")

	for _, part := range parts {
		dep := strings.TrimSpace(part)
		if dep != "" && dep != "None" {
			dependencies = append(dependencies, dep)
		}
	}

	return dependencies
}

// Helper function to read file content
func readFileContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}