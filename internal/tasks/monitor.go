package tasks

import (
	"fmt"
	"sync"
	"time"

	"github.com/dphaener/shiki-cli/internal/broker"
)

// ComplianceMonitor tracks agent compliance with task progress tracking requirements
type ComplianceMonitor struct {
	progressTracker     *ProgressTracker
	detector            *ComplianceDetector
	eventBroker         *broker.Broker
	sessionID           string

	// Monitoring state
	lastAgentUpdate     time.Time
	missedUpdatesCount  int
	complianceHistory   []ComplianceResult
	metrics             *ComplianceMetrics

	// Configuration
	complianceThreshold time.Duration // Max time between updates
	maxComplianceHistory int          // Max compliance results to keep

	// Thread safety
	mutex sync.RWMutex
}

// NewComplianceMonitor creates a new compliance monitor
func NewComplianceMonitor(tracker *ProgressTracker, broker *broker.Broker, sessionID string) *ComplianceMonitor {
	return &ComplianceMonitor{
		progressTracker:      tracker,
		detector:             NewComplianceDetector(),
		eventBroker:          broker,
		sessionID:           sessionID,
		complianceThreshold: 30 * time.Second,
		maxComplianceHistory: 50,
		complianceHistory:   make([]ComplianceResult, 0),
		metrics: &ComplianceMetrics{
			PatternSuccess:    make(map[string]int),
			RecentScores:      make([]float64, 0, 10),
			ImprovementTrend:  0.0,
		},
		lastAgentUpdate: time.Now(),
	}
}

// CheckCompliance analyzes agent response content for compliance
func (cm *ComplianceMonitor) CheckCompliance(responseContent string) ComplianceResult {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Perform compliance check
	result := cm.detector.CheckCompliance(responseContent)

	// Update monitoring state
	cm.lastAgentUpdate = time.Now()
	cm.addComplianceResult(result)
	cm.updateMetrics(result)

	// Emit compliance event if needed
	if result.OverallScore < 0.7 || !result.RequiredPassed {
		cm.emitComplianceViolationEvent(result, responseContent)
	}

	return result
}

// DetectMissedUpdates identifies when agent fails to provide progress updates
func (cm *ComplianceMonitor) DetectMissedUpdates(responseContent string) []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var violations []string

	// Check time since last update
	timeSinceUpdate := time.Since(cm.lastAgentUpdate)
	if timeSinceUpdate > cm.complianceThreshold {
		violations = append(violations,
			"Agent has not provided task progress updates within required timeframe")
		cm.missedUpdatesCount++
	}

	// Check for task completion without progress updates
	taskRefs := cm.detector.extractTaskIDs(responseContent)
	for _, taskID := range taskRefs {
		task, exists := cm.progressTracker.GetTask(taskID)
		if exists && task.Status == TaskStatePending {
			// Task mentioned but still pending - might need progress update
			violations = append(violations,
				"Task "+taskID+" mentioned but progress file not updated")
		}
	}

	return violations
}

// GenerateComplianceReport creates a detailed compliance report
func (cm *ComplianceMonitor) GenerateComplianceReport() ComplianceReport {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	report := ComplianceReport{
		SessionID:           cm.sessionID,
		GeneratedAt:        time.Now(),
		OverallScore:       cm.metrics.AverageScore,
		ComplianceLevel:    cm.determineOverallLevel(),
		TotalChecks:        cm.metrics.TotalChecks,
		PassedChecks:       cm.metrics.PassedChecks,
		MissedUpdates:      cm.missedUpdatesCount,
		RecentResults:      cm.getRecentResults(5),
		Recommendations:    cm.generateRecommendations(),
		ImprovementTrend:   cm.metrics.ImprovementTrend,
	}

	return report
}

// GetComplianceMetrics returns current compliance metrics
func (cm *ComplianceMonitor) GetComplianceMetrics() ComplianceMetrics {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Return a copy
	metrics := *cm.metrics
	metrics.RecentScores = append([]float64{}, cm.metrics.RecentScores...)
	metrics.PatternSuccess = make(map[string]int)
	for k, v := range cm.metrics.PatternSuccess {
		metrics.PatternSuccess[k] = v
	}

	return metrics
}

// SetComplianceThreshold updates the compliance threshold
func (cm *ComplianceMonitor) SetComplianceThreshold(threshold time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.complianceThreshold = threshold
}

// GetLastAgentUpdate returns when the agent last provided an update
func (cm *ComplianceMonitor) GetLastAgentUpdate() time.Time {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.lastAgentUpdate
}

// IsAgentCompliant returns whether the agent is currently compliant
func (cm *ComplianceMonitor) IsAgentCompliant() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Check time threshold
	if time.Since(cm.lastAgentUpdate) > cm.complianceThreshold {
		return false
	}

	// Check recent compliance scores
	if len(cm.metrics.RecentScores) > 0 {
		recentAvg := cm.calculateRecentAverage()
		return recentAvg >= 0.7
	}

	return true
}

// addComplianceResult adds a compliance result to history
func (cm *ComplianceMonitor) addComplianceResult(result ComplianceResult) {
	cm.complianceHistory = append(cm.complianceHistory, result)

	// Trim history if too large
	if len(cm.complianceHistory) > cm.maxComplianceHistory {
		cm.complianceHistory = cm.complianceHistory[1:]
	}
}

// updateMetrics updates compliance metrics
func (cm *ComplianceMonitor) updateMetrics(result ComplianceResult) {
	cm.metrics.TotalChecks++
	if result.RequiredPassed && result.OverallScore >= 0.7 {
		cm.metrics.PassedChecks++
	}

	// Update recent scores
	cm.metrics.RecentScores = append(cm.metrics.RecentScores, result.OverallScore)
	if len(cm.metrics.RecentScores) > 10 {
		cm.metrics.RecentScores = cm.metrics.RecentScores[1:]
	}

	// Update average score
	if cm.metrics.TotalChecks > 0 {
		cm.metrics.AverageScore = float64(cm.metrics.PassedChecks) / float64(cm.metrics.TotalChecks)
	}

	// Update pattern success counts
	for patternName, patternResult := range result.PatternResults {
		if patternResult.Matched {
			cm.metrics.PatternSuccess[patternName]++
		}
	}

	// Calculate improvement trend
	cm.metrics.ImprovementTrend = cm.calculateImprovementTrend()
	cm.metrics.LastCheck = time.Now()
}

// calculateRecentAverage calculates average of recent scores
func (cm *ComplianceMonitor) calculateRecentAverage() float64 {
	if len(cm.metrics.RecentScores) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, score := range cm.metrics.RecentScores {
		sum += score
	}
	return sum / float64(len(cm.metrics.RecentScores))
}

// calculateImprovementTrend calculates trend over recent scores
func (cm *ComplianceMonitor) calculateImprovementTrend() float64 {
	scores := cm.metrics.RecentScores
	if len(scores) < 3 {
		return 0.0
	}

	// Simple linear trend calculation
	n := len(scores)
	firstHalf := scores[:n/2]
	secondHalf := scores[n/2:]

	firstAvg := 0.0
	for _, score := range firstHalf {
		firstAvg += score
	}
	firstAvg /= float64(len(firstHalf))

	secondAvg := 0.0
	for _, score := range secondHalf {
		secondAvg += score
	}
	secondAvg /= float64(len(secondHalf))

	return secondAvg - firstAvg
}

// determineOverallLevel determines overall compliance level
func (cm *ComplianceMonitor) determineOverallLevel() ComplianceLevel {
	score := cm.metrics.AverageScore
	switch {
	case score >= 0.9:
		return ComplianceLevelExcellent
	case score >= 0.7:
		return ComplianceLevelGood
	case score >= 0.5:
		return ComplianceLevelFair
	default:
		return ComplianceLevelPoor
	}
}

// getRecentResults returns the most recent compliance results
func (cm *ComplianceMonitor) getRecentResults(count int) []ComplianceResult {
	if len(cm.complianceHistory) == 0 {
		return []ComplianceResult{}
	}

	start := len(cm.complianceHistory) - count
	if start < 0 {
		start = 0
	}

	results := make([]ComplianceResult, len(cm.complianceHistory[start:]))
	copy(results, cm.complianceHistory[start:])
	return results
}

// generateRecommendations creates recommendations based on compliance history
func (cm *ComplianceMonitor) generateRecommendations() []string {
	var recommendations []string

	// Check overall compliance rate
	if cm.metrics.AverageScore < 0.7 {
		recommendations = append(recommendations,
			"Improve overall compliance by following task progress tracking requirements more consistently")
	}

	// Check missed updates
	if cm.missedUpdatesCount > 3 {
		recommendations = append(recommendations,
			"Provide more frequent progress updates - aim for updates every 30 seconds during active work")
	}

	// Check improvement trend
	if cm.metrics.ImprovementTrend < -0.1 {
		recommendations = append(recommendations,
			"Compliance is declining - review the implementation prompt requirements")
	}

	// Check pattern success rates
	for patternName, successCount := range cm.metrics.PatternSuccess {
		successRate := float64(successCount) / float64(cm.metrics.TotalChecks)
		if successRate < 0.5 {
			switch patternName {
			case "TaskStartAnnouncement":
				recommendations = append(recommendations,
					"Use explicit 'Starting T###' announcements when beginning tasks")
			case "TaskCompletionAnnouncement":
				recommendations = append(recommendations,
					"Use explicit 'Completed T### successfully' announcements when finishing tasks")
			case "ProgressFileCreation":
				recommendations = append(recommendations,
					"Always announce when creating or updating the task-progress.md file")
			}
		}
	}

	return recommendations
}

// emitComplianceViolationEvent emits an event when compliance violations occur
func (cm *ComplianceMonitor) emitComplianceViolationEvent(result ComplianceResult, content string) {
	// In a full implementation, this would emit a broker event
	// For now, we'll store the violation
}

// ComplianceReport represents a comprehensive compliance report
type ComplianceReport struct {
	SessionID           string             `json:"session_id"`
	GeneratedAt        time.Time          `json:"generated_at"`
	OverallScore       float64            `json:"overall_score"`
	ComplianceLevel    ComplianceLevel    `json:"compliance_level"`
	TotalChecks        int                `json:"total_checks"`
	PassedChecks       int                `json:"passed_checks"`
	MissedUpdates      int                `json:"missed_updates"`
	RecentResults      []ComplianceResult `json:"recent_results"`
	Recommendations    []string           `json:"recommendations"`
	ImprovementTrend   float64            `json:"improvement_trend"`
}

// GetComplianceRate returns the compliance rate as a percentage
func (cr *ComplianceReport) GetComplianceRate() float64 {
	if cr.TotalChecks == 0 {
		return 0.0
	}
	return float64(cr.PassedChecks) / float64(cr.TotalChecks) * 100
}

// IsImproving returns whether compliance is improving
func (cr *ComplianceReport) IsImproving() bool {
	return cr.ImprovementTrend > 0.05
}

// GetSummary returns a text summary of compliance status
func (cr *ComplianceReport) GetSummary() string {
	rate := cr.GetComplianceRate()
	trend := "stable"
	if cr.ImprovementTrend > 0.05 {
		trend = "improving"
	} else if cr.ImprovementTrend < -0.05 {
		trend = "declining"
	}

	return fmt.Sprintf("Compliance: %.1f%% (%s) - %d/%d checks passed - %s",
		rate, cr.ComplianceLevel, cr.PassedChecks, cr.TotalChecks, trend)
}