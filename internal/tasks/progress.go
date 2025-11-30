package tasks

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// TaskProgressState represents the current status of a task
type TaskProgressState string

const (
	TaskStatePending     TaskProgressState = "pending"
	TaskStateInProgress  TaskProgressState = "in_progress"
	TaskStateCompleted   TaskProgressState = "completed"
	TaskStateBlocked     TaskProgressState = "blocked"
	TaskStateSkipped     TaskProgressState = "skipped"
)

// String returns the string representation of the task state
func (tps TaskProgressState) String() string {
	return string(tps)
}

// IsValid checks if the task progress state is valid
func (tps TaskProgressState) IsValid() bool {
	switch tps {
	case TaskStatePending, TaskStateInProgress, TaskStateCompleted, TaskStateBlocked, TaskStateSkipped:
		return true
	default:
		return false
	}
}

// TaskProgress represents the progress of a single task
type TaskProgress struct {
	ID          string            `json:"id"`          // e.g., "T001"
	Description string            `json:"description"` // Task description
	Status      TaskProgressState `json:"status"`      // Current status
	StartedAt   *time.Time        `json:"started_at"`  // When task was started
	CompletedAt *time.Time        `json:"completed_at"` // When task was completed
	BlockReason string            `json:"block_reason"` // Reason if blocked
	Output      string            `json:"output"`      // Any output or notes
}

// WorkPackageProgress represents the progress of a work package
type WorkPackageProgress struct {
	ID          string            `json:"id"`          // e.g., "WP01"
	Name        string            `json:"name"`        // Work package name
	Priority    string            `json:"priority"`    // Priority level
	Status      TaskProgressState `json:"status"`      // Aggregated status
	StartedAt   *time.Time        `json:"started_at"`  // When any task was started
	CompletedAt *time.Time        `json:"completed_at"` // When all tasks completed
	Tasks       []TaskProgress    `json:"tasks"`       // Individual task progress
	Goal        string            `json:"goal"`        // Goal description
}

// ProgressSummary provides an overview of the entire implementation progress
type ProgressSummary struct {
	TotalWorkPackages     int     `json:"total_work_packages"`
	CompletedWorkPackages int     `json:"completed_work_packages"`
	TotalTasks            int     `json:"total_tasks"`
	CompletedTasks        int     `json:"completed_tasks"`
	InProgressTasks       int     `json:"in_progress_tasks"`
	BlockedTasks          int     `json:"blocked_tasks"`
	SkippedTasks          int     `json:"skipped_tasks"`
	OverallProgress       float64 `json:"overall_progress"`
	LastUpdated           time.Time `json:"last_updated"`
}

// ProgressTracker manages task and work package progress in memory
type ProgressTracker struct {
	taskStructure    *TaskStructure
	workPackages     map[string]*WorkPackageProgress
	progressFile     string
	lastUpdated      time.Time
	mutex            sync.RWMutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(taskStructure *TaskStructure, progressFile string) *ProgressTracker {
	pt := &ProgressTracker{
		taskStructure: taskStructure,
		workPackages:  make(map[string]*WorkPackageProgress),
		progressFile:  progressFile,
		lastUpdated:   time.Now(),
	}

	// Initialize work packages from task structure
	pt.initializeFromTaskStructure()

	return pt
}

// initializeFromTaskStructure initializes progress tracking from the parsed task structure
func (pt *ProgressTracker) initializeFromTaskStructure() {
	for _, wp := range pt.taskStructure.WorkPackages {
		wpProgress := &WorkPackageProgress{
			ID:       wp.ID,
			Name:     wp.Name,
			Priority: wp.Priority,
			Status:   TaskStatePending,
			Goal:     wp.Goal,
			Tasks:    make([]TaskProgress, 0, len(wp.Tasks)),
		}

		// Initialize task progress
		for _, task := range wp.Tasks {
			taskProgress := TaskProgress{
				ID:          task.ID,
				Description: task.Description,
				Status:      TaskStatePending,
			}
			wpProgress.Tasks = append(wpProgress.Tasks, taskProgress)
		}

		pt.workPackages[wp.ID] = wpProgress
	}
}

// UpdateTaskStatus updates the status of a specific task
func (pt *ProgressTracker) UpdateTaskStatus(taskID string, status TaskProgressState, reason string) error {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	// Find the work package containing this task
	var targetWP *WorkPackageProgress
	var taskIndex int = -1

	for _, wp := range pt.workPackages {
		for i, task := range wp.Tasks {
			if task.ID == taskID {
				targetWP = wp
				taskIndex = i
				break
			}
		}
		if targetWP != nil {
			break
		}
	}

	if targetWP == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Update task status
	task := &targetWP.Tasks[taskIndex]
	oldStatus := task.Status
	task.Status = status

	now := time.Now()
	pt.lastUpdated = now

	// Update timestamps based on status change
	switch status {
	case TaskStateInProgress:
		if oldStatus == TaskStatePending {
			task.StartedAt = &now
		}
		// Clear completion time if moving back to in progress
		task.CompletedAt = nil
	case TaskStateCompleted:
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
		task.CompletedAt = &now
	case TaskStateBlocked:
		task.BlockReason = reason
	case TaskStateSkipped:
		task.CompletedAt = &now
		if reason != "" {
			task.BlockReason = reason
		}
	}

	// Update work package status and timestamps
	pt.updateWorkPackageStatus(targetWP)

	return nil
}

// updateWorkPackageStatus updates the aggregated status of a work package
func (pt *ProgressTracker) updateWorkPackageStatus(wp *WorkPackageProgress) {
	var pending, inProgress, completed, blocked, skipped int

	for _, task := range wp.Tasks {
		switch task.Status {
		case TaskStatePending:
			pending++
		case TaskStateInProgress:
			inProgress++
		case TaskStateCompleted:
			completed++
		case TaskStateBlocked:
			blocked++
		case TaskStateSkipped:
			skipped++
		}
	}

	totalTasks := len(wp.Tasks)
	oldStatus := wp.Status

	// Determine work package status
	if completed+skipped == totalTasks {
		wp.Status = TaskStateCompleted
		if wp.CompletedAt == nil {
			now := time.Now()
			wp.CompletedAt = &now
		}
	} else if blocked > 0 && inProgress == 0 {
		wp.Status = TaskStateBlocked
	} else if inProgress > 0 {
		wp.Status = TaskStateInProgress
		if oldStatus == TaskStatePending {
			now := time.Now()
			wp.StartedAt = &now
		}
	} else {
		wp.Status = TaskStatePending
	}
}

// GetProgress returns a summary of overall progress
func (pt *ProgressTracker) GetProgress() ProgressSummary {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	var totalTasks, completedTasks, inProgressTasks, blockedTasks, skippedTasks int
	var completedWPs int

	for _, wp := range pt.workPackages {
		if wp.Status == TaskStateCompleted {
			completedWPs++
		}

		for _, task := range wp.Tasks {
			totalTasks++
			switch task.Status {
			case TaskStateCompleted:
				completedTasks++
			case TaskStateInProgress:
				inProgressTasks++
			case TaskStateBlocked:
				blockedTasks++
			case TaskStateSkipped:
				skippedTasks++
			}
		}
	}

	overallProgress := 0.0
	if totalTasks > 0 {
		overallProgress = float64(completedTasks+skippedTasks) / float64(totalTasks) * 100
	}

	return ProgressSummary{
		TotalWorkPackages:     len(pt.workPackages),
		CompletedWorkPackages: completedWPs,
		TotalTasks:            totalTasks,
		CompletedTasks:        completedTasks,
		InProgressTasks:       inProgressTasks,
		BlockedTasks:          blockedTasks,
		SkippedTasks:          skippedTasks,
		OverallProgress:       overallProgress,
		LastUpdated:           pt.lastUpdated,
	}
}

// GetWorkPackages returns all work package progress (sorted by ID)
func (pt *ProgressTracker) GetWorkPackages() []WorkPackageProgress {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	var workPackages []WorkPackageProgress
	for _, wp := range pt.workPackages {
		workPackages = append(workPackages, *wp)
	}

	// Sort by ID for consistent ordering
	sort.Slice(workPackages, func(i, j int) bool {
		return workPackages[i].ID < workPackages[j].ID
	})

	return workPackages
}

// GetWorkPackage returns a specific work package progress
func (pt *ProgressTracker) GetWorkPackage(wpID string) (*WorkPackageProgress, bool) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	wp, exists := pt.workPackages[wpID]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	wpCopy := *wp
	return &wpCopy, true
}

// GetTask returns a specific task progress
func (pt *ProgressTracker) GetTask(taskID string) (*TaskProgress, bool) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	for _, wp := range pt.workPackages {
		for _, task := range wp.Tasks {
			if task.ID == taskID {
				// Return a copy to prevent external modification
				taskCopy := task
				return &taskCopy, true
			}
		}
	}

	return nil, false
}

// AddDiscoveredTask adds a new task that was discovered during implementation
func (pt *ProgressTracker) AddDiscoveredTask(workPackageID, taskID, description string) error {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	wp, exists := pt.workPackages[workPackageID]
	if !exists {
		return fmt.Errorf("work package %s not found", workPackageID)
	}

	// Check if task already exists
	for _, task := range wp.Tasks {
		if task.ID == taskID {
			return fmt.Errorf("task %s already exists in work package %s", taskID, workPackageID)
		}
	}

	// Add new task
	newTask := TaskProgress{
		ID:          taskID,
		Description: description,
		Status:      TaskStatePending,
	}

	wp.Tasks = append(wp.Tasks, newTask)
	pt.lastUpdated = time.Now()

	return nil
}

// MarkTaskAsSkipped marks a task as skipped with a reason
func (pt *ProgressTracker) MarkTaskAsSkipped(taskID, reason string) error {
	return pt.UpdateTaskStatus(taskID, TaskStateSkipped, reason)
}

// SetTaskOutput sets output/notes for a task
func (pt *ProgressTracker) SetTaskOutput(taskID, output string) error {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	for _, wp := range pt.workPackages {
		for i, task := range wp.Tasks {
			if task.ID == taskID {
				wp.Tasks[i].Output = output
				pt.lastUpdated = time.Now()
				return nil
			}
		}
	}

	return fmt.Errorf("task %s not found", taskID)
}