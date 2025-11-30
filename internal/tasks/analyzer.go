package tasks

import (
	"regexp"
	"strings"
	"sync"
)

// TaskStatusUpdate represents a detected task status change
type TaskStatusUpdate struct {
	TaskID    string            // e.g., "T001"
	Status    TaskProgressState // New status
	Reason    string            // Reason for status change (for blocked/skipped)
	Confidence float64         // Confidence level 0.0-1.0
}

// ResponseAnalyzer analyzes agent responses to detect task completion indicators
type ResponseAnalyzer struct {
	taskStructure *TaskStructure
	keywords      map[TaskProgressState][]string // Keywords that indicate each status
	patterns      map[TaskProgressState][]*regexp.Regexp // Regex patterns for status detection
	mutex         sync.RWMutex
}

// NewResponseAnalyzer creates a new response analyzer
func NewResponseAnalyzer(taskStructure *TaskStructure) *ResponseAnalyzer {
	ra := &ResponseAnalyzer{
		taskStructure: taskStructure,
		keywords:      make(map[TaskProgressState][]string),
		patterns:      make(map[TaskProgressState][]*regexp.Regexp),
	}

	ra.initializeKeywords()
	ra.initializePatterns()

	return ra
}

// initializeKeywords sets up keyword mappings for status detection
func (ra *ResponseAnalyzer) initializeKeywords() {
	ra.keywords[TaskStateInProgress] = []string{
		"starting", "started", "beginning", "working on", "in progress", "implementing",
		"creating", "building", "developing", "setting up", "initializing",
	}

	ra.keywords[TaskStateCompleted] = []string{
		"completed", "finished", "done", "complete", "successfully", "accomplished",
		"implemented", "created", "built", "established", "configured",
	}

	ra.keywords[TaskStateBlocked] = []string{
		"blocked", "stuck", "error", "failed", "unable to", "cannot", "missing",
		"need", "requires", "waiting for", "dependency",
	}

	ra.keywords[TaskStateSkipped] = []string{
		"skipped", "skipping", "not needed", "unnecessary", "out of scope",
		"not required", "removing", "excluding",
	}
}

// initializePatterns sets up regex patterns for more sophisticated detection
func (ra *ResponseAnalyzer) initializePatterns() {
	// Patterns for task ID references with status indicators
	ra.patterns[TaskStateInProgress] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(starting|beginning|working on|implementing)\s+(T\d+)`),
		regexp.MustCompile(`(?i)(T\d+)\s+(is\s+)?(in progress|started|beginning)`),
		regexp.MustCompile(`(?i)now\s+(working on|implementing|starting)\s+(T\d+)`),
	}

	ra.patterns[TaskStateCompleted] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(completed|finished|done with)\s+(T\d+)`),
		regexp.MustCompile(`(?i)(T\d+)\s+(is\s+)?(completed?|finished|done|successfully implemented)`),
		regexp.MustCompile(`(?i)(T\d+)\s+successfully\s+(completed|finished|implemented|created)`),
		regexp.MustCompile(`(?i)✓\s*(T\d+)`), // Checkmark followed by task ID
	}

	ra.patterns[TaskStateBlocked] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(T\d+)\s+(is\s+)?(blocked|stuck|failed)`),
		regexp.MustCompile(`(?i)(blocked on|stuck on|error with|failed on)\s+(T\d+)`),
		regexp.MustCompile(`(?i)(T\d+)\s+blocked:?\s+(.+)`),
		regexp.MustCompile(`(?i)cannot\s+(complete|continue|proceed with)\s+(T\d+)`),
	}

	ra.patterns[TaskStateSkipped] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(skipped|skipping)\s+(T\d+)`),
		regexp.MustCompile(`(?i)(T\d+)\s+(is\s+)?(skipped|not needed|unnecessary)`),
		regexp.MustCompile(`(?i)(T\d+)\s+skipped:?\s+(.+)`),
	}
}

// AnalyzeResponse analyzes agent response content for task status updates
func (ra *ResponseAnalyzer) AnalyzeResponse(content string) []TaskStatusUpdate {
	ra.mutex.RLock()
	defer ra.mutex.RUnlock()

	var updates []TaskStatusUpdate

	// First, find explicit task references with status patterns
	explicitUpdates := ra.findExplicitTaskUpdates(content)
	updates = append(updates, explicitUpdates...)

	// Then, find implicit task references based on keywords and context
	implicitUpdates := ra.findImplicitTaskUpdates(content)
	updates = append(updates, implicitUpdates...)

	// Remove duplicates and prioritize higher confidence updates
	return ra.dedupAndPrioritizeUpdates(updates)
}

// findExplicitTaskUpdates finds task updates using regex patterns
func (ra *ResponseAnalyzer) findExplicitTaskUpdates(content string) []TaskStatusUpdate {
	var updates []TaskStatusUpdate

	for status, patterns := range ra.patterns {
		for _, pattern := range patterns {
			matches := pattern.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				taskID := ra.extractTaskIDFromMatch(match)
				if taskID != "" && ra.isValidTaskID(taskID) {
					reason := ra.extractReasonFromMatch(match, status)
					confidence := 0.9 // High confidence for pattern matches

					updates = append(updates, TaskStatusUpdate{
						TaskID:     taskID,
						Status:     status,
						Reason:     reason,
						Confidence: confidence,
					})
				}
			}
		}
	}

	return updates
}

// findImplicitTaskUpdates finds task updates based on keywords and context
func (ra *ResponseAnalyzer) findImplicitTaskUpdates(content string) []TaskStatusUpdate {
	var updates []TaskStatusUpdate

	// Find all task ID references in the content
	taskIDs := ra.findTaskReferences(content)

	for _, taskID := range taskIDs {
		// Look for status keywords in the vicinity of the task ID
		taskContext := ra.extractTaskContext(content, taskID)

		for status, keywords := range ra.keywords {
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(taskContext), strings.ToLower(keyword)) {
					confidence := ra.calculateKeywordConfidence(taskContext, keyword, status)
					if confidence > 0.5 { // Only include if confidence is reasonable
						updates = append(updates, TaskStatusUpdate{
							TaskID:     taskID,
							Status:     status,
							Reason:     ra.extractContextualReason(taskContext, status),
							Confidence: confidence,
						})
					}
				}
			}
		}
	}

	return updates
}

// DetectTaskReferences finds all task ID references in content
func (ra *ResponseAnalyzer) DetectTaskReferences(content string) []string {
	return ra.findTaskReferences(content)
}

// findTaskReferences extracts task IDs from content
func (ra *ResponseAnalyzer) findTaskReferences(content string) []string {
	taskPattern := regexp.MustCompile(`\b(T\d+)\b`)
	matches := taskPattern.FindAllStringSubmatch(content, -1)

	var taskIDs []string
	seen := make(map[string]bool)

	for _, match := range matches {
		taskID := match[1]
		if ra.isValidTaskID(taskID) && !seen[taskID] {
			taskIDs = append(taskIDs, taskID)
			seen[taskID] = true
		}
	}

	return taskIDs
}

// InferTaskStatus infers task status from content and task ID context
func (ra *ResponseAnalyzer) InferTaskStatus(content, taskID string) TaskProgressState {
	if !ra.isValidTaskID(taskID) {
		return TaskStatePending
	}

	updates := ra.AnalyzeResponse(content)

	// Find the highest confidence update for this task
	var bestUpdate *TaskStatusUpdate
	for _, update := range updates {
		if update.TaskID == taskID {
			if bestUpdate == nil || update.Confidence > bestUpdate.Confidence {
				bestUpdate = &update
			}
		}
	}

	if bestUpdate != nil {
		return bestUpdate.Status
	}

	return TaskStatePending
}

// extractTaskIDFromMatch extracts task ID from regex match
func (ra *ResponseAnalyzer) extractTaskIDFromMatch(match []string) string {
	taskPattern := regexp.MustCompile(`T\d+`)
	for _, group := range match {
		if taskPattern.MatchString(group) {
			return group
		}
	}
	return ""
}

// extractReasonFromMatch extracts reason from regex match for blocked/skipped tasks
func (ra *ResponseAnalyzer) extractReasonFromMatch(match []string, status TaskProgressState) string {
	if status != TaskStateBlocked && status != TaskStateSkipped {
		return ""
	}

	// For patterns that capture reasons (usually the last group)
	if len(match) > 2 {
		reason := strings.TrimSpace(match[len(match)-1])
		// Only return if it looks like a reason (not just the task ID or status)
		if len(reason) > 10 && !regexp.MustCompile(`^T\d+$`).MatchString(reason) {
			return reason
		}
	}

	return ""
}

// extractTaskContext extracts surrounding context for a task ID
func (ra *ResponseAnalyzer) extractTaskContext(content, taskID string) string {
	taskIndex := strings.Index(strings.ToLower(content), strings.ToLower(taskID))
	if taskIndex == -1 {
		return ""
	}

	// Extract context around the task ID (200 characters before and after)
	start := taskIndex - 200
	if start < 0 {
		start = 0
	}

	end := taskIndex + len(taskID) + 200
	if end > len(content) {
		end = len(content)
	}

	return content[start:end]
}

// calculateKeywordConfidence calculates confidence based on keyword presence and context
func (ra *ResponseAnalyzer) calculateKeywordConfidence(context, keyword string, status TaskProgressState) float64 {
	lowerContext := strings.ToLower(context)
	lowerKeyword := strings.ToLower(keyword)

	// Base confidence for keyword presence
	confidence := 0.6

	// Boost confidence for multiple occurrences
	occurrences := strings.Count(lowerContext, lowerKeyword)
	if occurrences > 1 {
		confidence += float64(occurrences-1) * 0.1
	}

	// Boost confidence for certain combinations
	switch status {
	case TaskStateCompleted:
		if strings.Contains(lowerContext, "successfully") ||
		   strings.Contains(lowerContext, "✓") ||
		   strings.Contains(lowerContext, "finished") {
			confidence += 0.2
		}
	case TaskStateBlocked:
		if strings.Contains(lowerContext, "error") ||
		   strings.Contains(lowerContext, "failed") ||
		   strings.Contains(lowerContext, "cannot") {
			confidence += 0.2
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// extractContextualReason extracts reason from context for blocked/skipped tasks
func (ra *ResponseAnalyzer) extractContextualReason(context string, status TaskProgressState) string {
	if status != TaskStateBlocked && status != TaskStateSkipped {
		return ""
	}

	// Look for common reason patterns
	reasonPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)because\s+(.{10,100})`),
		regexp.MustCompile(`(?i)due to\s+(.{10,100})`),
		regexp.MustCompile(`(?i)reason:\s*(.{10,100})`),
		regexp.MustCompile(`(?i)error:\s*(.{10,100})`),
	}

	for _, pattern := range reasonPatterns {
		match := pattern.FindStringSubmatch(context)
		if match != nil && len(match) > 1 {
			reason := strings.TrimSpace(match[1])
			// Clean up the reason (remove newlines, extra spaces)
			reason = regexp.MustCompile(`\s+`).ReplaceAllString(reason, " ")
			if len(reason) > 10 && len(reason) < 200 {
				return reason
			}
		}
	}

	return ""
}

// isValidTaskID checks if a task ID exists in the task structure
func (ra *ResponseAnalyzer) isValidTaskID(taskID string) bool {
	_, exists := ra.taskStructure.TaskMap[taskID]
	return exists
}

// dedupAndPrioritizeUpdates removes duplicates and keeps highest confidence updates
func (ra *ResponseAnalyzer) dedupAndPrioritizeUpdates(updates []TaskStatusUpdate) []TaskStatusUpdate {
	if len(updates) == 0 {
		return updates
	}

	// Group by task ID and keep highest confidence
	bestUpdates := make(map[string]TaskStatusUpdate)

	for _, update := range updates {
		existing, exists := bestUpdates[update.TaskID]
		if !exists || update.Confidence > existing.Confidence {
			bestUpdates[update.TaskID] = update
		}
	}

	// Convert back to slice
	var result []TaskStatusUpdate
	for _, update := range bestUpdates {
		result = append(result, update)
	}

	return result
}

// GetKeywords returns the keyword mappings for testing/debugging
func (ra *ResponseAnalyzer) GetKeywords() map[TaskProgressState][]string {
	ra.mutex.RLock()
	defer ra.mutex.RUnlock()

	// Return a copy
	keywords := make(map[TaskProgressState][]string)
	for status, words := range ra.keywords {
		keywords[status] = append([]string{}, words...)
	}

	return keywords
}

// AddCustomKeywords allows adding custom keywords for status detection
func (ra *ResponseAnalyzer) AddCustomKeywords(status TaskProgressState, keywords []string) {
	ra.mutex.Lock()
	defer ra.mutex.Unlock()

	existing := ra.keywords[status]
	for _, keyword := range keywords {
		// Check if keyword already exists
		found := false
		for _, existing := range existing {
			if strings.EqualFold(existing, keyword) {
				found = true
				break
			}
		}
		if !found {
			ra.keywords[status] = append(ra.keywords[status], keyword)
		}
	}
}