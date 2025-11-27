package components

import (
	"strings"
)

// GenerateLoadingMessage creates a contextual loading message based on the last user message and current phase.
func GenerateLoadingMessage(lastUserMessage string, phase string) string {
	if lastUserMessage == "" {
		return getDefaultPhaseMessage(phase)
	}

	lower := strings.ToLower(lastUserMessage)

	// Check for question patterns
	if isQuestion(lower) {
		return "Researching your question..."
	}

	// Check for feedback/change patterns
	if containsAny(lower, []string{"change", "update", "modify", "fix", "adjust", "revise", "instead"}) {
		return "Incorporating your feedback..."
	}

	// Check for approval/confirmation patterns
	if containsAny(lower, []string{"yes", "ok", "sure", "go ahead", "looks good", "approve", "accept"}) {
		return "Processing your approval..."
	}

	// Check for specific topic mentions and extract a key topic
	if topic := extractTopic(lower); topic != "" {
		return "Thinking about " + topic + "..."
	}

	// Fall back to phase-specific default
	return getDefaultPhaseMessage(phase)
}

// getDefaultPhaseMessage returns a default message for the given workflow phase.
func getDefaultPhaseMessage(phase string) string {
	switch phase {
	case "specify":
		return "Discovering requirements..."
	case "plan":
		return "Creating implementation plan..."
	case "tasks":
		return "Generating task breakdown..."
	case "implement":
		return "Writing implementation..."
	default:
		return "Thinking..."
	}
}

// isQuestion checks if the message appears to be a question.
func isQuestion(text string) bool {
	// Check for question mark
	if strings.Contains(text, "?") {
		return true
	}

	// Check for question words at the start
	questionStarters := []string{"what", "why", "how", "when", "where", "who", "which", "can", "could", "would", "should", "is", "are", "do", "does"}
	for _, starter := range questionStarters {
		if strings.HasPrefix(text, starter+" ") {
			return true
		}
	}

	return false
}

// containsAny checks if text contains any of the given substrings.
func containsAny(text string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

// extractTopic tries to extract a meaningful topic from the message.
func extractTopic(text string) string {
	// Common feature/technical topics
	topics := map[string]string{
		"auth":           "authentication",
		"login":          "authentication",
		"user":           "user management",
		"database":       "database",
		"api":            "API design",
		"endpoint":       "API endpoints",
		"test":           "testing",
		"error":          "error handling",
		"validation":     "validation",
		"security":       "security",
		"performance":    "performance",
		"cache":          "caching",
		"config":         "configuration",
		"deploy":         "deployment",
		"docker":         "containerization",
		"notification":   "notifications",
		"email":          "email",
		"payment":        "payments",
		"search":         "search",
		"filter":         "filtering",
		"sort":           "sorting",
		"pagination":     "pagination",
		"upload":         "file uploads",
		"download":       "downloads",
		"export":         "data export",
		"import":         "data import",
		"webhook":        "webhooks",
		"integration":    "integrations",
		"migration":      "migration",
		"refactor":       "refactoring",
		"ui":             "user interface",
		"ux":             "user experience",
		"frontend":       "frontend",
		"backend":        "backend",
		"middleware":     "middleware",
		"logging":        "logging",
		"monitoring":     "monitoring",
		"analytics":      "analytics",
		"dashboard":      "dashboard",
		"report":         "reporting",
		"permission":     "permissions",
		"role":           "roles",
		"access":         "access control",
	}

	for keyword, topic := range topics {
		if strings.Contains(text, keyword) {
			return topic
		}
	}

	return ""
}
