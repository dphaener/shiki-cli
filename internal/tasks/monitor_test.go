package tasks

import (
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/internal/broker"
)

func TestComplianceMonitor_Basic(t *testing.T) {
	// Create test infrastructure
	taskStructure := createTestTaskStructure()
	taskStructure.TaskMap = map[string]*Task{
		"T001": &taskStructure.WorkPackages[0].Tasks[0],
		"T002": &taskStructure.WorkPackages[0].Tasks[1],
		"T003": &taskStructure.WorkPackages[0].Tasks[2],
	}
	taskStructure.WPMap = map[string]*WorkPackage{
		"WP01": &taskStructure.WorkPackages[0],
		"WP02": &taskStructure.WorkPackages[1],
	}

	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")
	eventBroker := broker.New()
	defer eventBroker.Shutdown()

	monitor := NewComplianceMonitor(tracker, eventBroker, "test-session")

	// Test basic compliance check
	goodResponse := "Starting T001: Initialize project structure"
	result := monitor.CheckCompliance(goodResponse)

	if result.OverallScore < 0.5 {
		t.Errorf("expected reasonable compliance score for good response, got %.2f", result.OverallScore)
	}

	if !result.PatternResults["TaskStartAnnouncement"].Matched {
		t.Error("expected to detect task start announcement")
	}
}

func TestComplianceMonitor_MissedUpdates(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")
	eventBroker := broker.New()
	defer eventBroker.Shutdown()

	monitor := NewComplianceMonitor(tracker, eventBroker, "test-session")

	// Set a short threshold for testing
	monitor.SetComplianceThreshold(1 * time.Millisecond)

	// Wait for threshold to pass
	time.Sleep(2 * time.Millisecond)

	violations := monitor.DetectMissedUpdates("Some response without task updates")

	if len(violations) == 0 {
		t.Error("expected to detect missed updates violation")
	}

	found := false
	for _, violation := range violations {
		if violation == "Agent has not provided task progress updates within required timeframe" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected specific missed updates violation message")
	}
}

func TestComplianceMonitor_Report(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")
	eventBroker := broker.New()
	defer eventBroker.Shutdown()

	monitor := NewComplianceMonitor(tracker, eventBroker, "test-session")

	// Add some compliance checks
	monitor.CheckCompliance("Starting T001: Initialize project")
	monitor.CheckCompliance("Completed T001 successfully")
	monitor.CheckCompliance("Some vague response")

	report := monitor.GenerateComplianceReport()

	if report.SessionID != "test-session" {
		t.Errorf("expected session ID 'test-session', got '%s'", report.SessionID)
	}

	if report.TotalChecks != 3 {
		t.Errorf("expected 3 total checks, got %d", report.TotalChecks)
	}

	if len(report.RecentResults) == 0 {
		t.Error("expected recent results in report")
	}
}

func TestComplianceMonitor_IsAgentCompliant(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")
	eventBroker := broker.New()
	defer eventBroker.Shutdown()

	monitor := NewComplianceMonitor(tracker, eventBroker, "test-session")

	// Should be compliant initially
	if !monitor.IsAgentCompliant() {
		t.Error("expected agent to be compliant initially")
	}

	// Add a good compliance check
	monitor.CheckCompliance("Starting T001: Initialize project structure")
	if !monitor.IsAgentCompliant() {
		t.Error("expected agent to remain compliant after good response")
	}

	// Set very short threshold and wait
	monitor.SetComplianceThreshold(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	// Should now be non-compliant due to time threshold
	if monitor.IsAgentCompliant() {
		t.Error("expected agent to be non-compliant after threshold exceeded")
	}
}

func TestComplianceMonitor_MetricsTracking(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")
	eventBroker := broker.New()
	defer eventBroker.Shutdown()

	monitor := NewComplianceMonitor(tracker, eventBroker, "test-session")

	// Add several compliance checks
	responses := []string{
		"Starting T001: Initialize project structure",
		"Completed T001 successfully",
		"T002 blocked: missing dependencies",
		"Some non-compliant response",
		"Starting T003: Create core functionality",
	}

	for _, response := range responses {
		monitor.CheckCompliance(response)
	}

	metrics := monitor.GetComplianceMetrics()

	if metrics.TotalChecks != 5 {
		t.Errorf("expected 5 total checks, got %d", metrics.TotalChecks)
	}

	if len(metrics.RecentScores) != 5 {
		t.Errorf("expected 5 recent scores, got %d", len(metrics.RecentScores))
	}

	if len(metrics.PatternSuccess) == 0 {
		t.Error("expected pattern success tracking")
	}
}

func TestComplianceReport_Methods(t *testing.T) {
	report := ComplianceReport{
		TotalChecks:     10,
		PassedChecks:    8,
		ImprovementTrend: 0.1,
	}

	rate := report.GetComplianceRate()
	if rate != 80.0 {
		t.Errorf("expected compliance rate 80.0, got %.1f", rate)
	}

	if !report.IsImproving() {
		t.Error("expected report to show improvement")
	}

	summary := report.GetSummary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}