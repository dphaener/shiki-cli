package agent

import (
	"time"
)

// Model pricing constants (per 1M tokens)
const (
	// Claude Sonnet 4 pricing
	SonnetInputPrice  = 3.00  // $3/1M input tokens
	SonnetOutputPrice = 15.00 // $15/1M output tokens

	// Claude Opus pricing
	OpusInputPrice  = 15.00 // $15/1M input tokens
	OpusOutputPrice = 75.00 // $75/1M output tokens

	// Claude Haiku pricing
	HaikuInputPrice  = 0.25 // $0.25/1M input tokens
	HaikuOutputPrice = 1.25 // $1.25/1M output tokens
)

// TurnResult contains metrics and outcome from a single agent turn
type TurnResult struct {
	AgentID       string
	TurnNumber    int
	Duration      time.Duration
	TokensUsed    int
	InputTokens   int
	OutputTokens  int
	Cost          float64
	Response      string
	ToolCalls     int
	Success       bool
	Error         string
	CacheHits     int    // For prompt caching
	CacheMisses   int    // For prompt caching
}

// CalculateCost computes the cost of a turn based on token usage and model
func CalculateCost(tokensUsed int, model string) float64 {
	// Simplified calculation assuming 50/50 input/output split
	// In production, actual input/output token counts should be used
	tokensMillions := float64(tokensUsed) / 1_000_000.0

	var avgPrice float64
	switch model {
	case "claude-sonnet-4", "claude-sonnet-4-5":
		avgPrice = (SonnetInputPrice + SonnetOutputPrice) / 2.0
	case "claude-opus-4":
		avgPrice = (OpusInputPrice + OpusOutputPrice) / 2.0
	case "claude-haiku-4":
		avgPrice = (HaikuInputPrice + HaikuOutputPrice) / 2.0
	default:
		// Default to Sonnet pricing for unknown models
		avgPrice = (SonnetInputPrice + SonnetOutputPrice) / 2.0
	}

	return tokensMillions * avgPrice
}

// CalculateCostDetailed computes cost using actual input/output token breakdown
func CalculateCostDetailed(inputTokens, outputTokens int, model string) float64 {
	inputMillions := float64(inputTokens) / 1_000_000.0
	outputMillions := float64(outputTokens) / 1_000_000.0

	var inputPrice, outputPrice float64
	switch model {
	case "claude-sonnet-4", "claude-sonnet-4-5":
		inputPrice = SonnetInputPrice
		outputPrice = SonnetOutputPrice
	case "claude-opus-4":
		inputPrice = OpusInputPrice
		outputPrice = OpusOutputPrice
	case "claude-haiku-4":
		inputPrice = HaikuInputPrice
		outputPrice = HaikuOutputPrice
	default:
		// Default to Sonnet pricing
		inputPrice = SonnetInputPrice
		outputPrice = SonnetOutputPrice
	}

	return (inputMillions * inputPrice) + (outputMillions * outputPrice)
}

// AgentMetrics tracks cumulative metrics for an agent across all turns
type AgentMetrics struct {
	AgentID      string
	TotalTurns   int
	TotalTokens  int
	InputTokens  int
	OutputTokens int
	TotalCost    float64
	TotalTime    time.Duration
	ToolCalls    int
	Errors       int
}

// UpdateMetrics adds a turn result to the cumulative metrics
func (m *AgentMetrics) UpdateMetrics(result *TurnResult) {
	m.TotalTurns++
	m.TotalTokens += result.TokensUsed
	m.InputTokens += result.InputTokens
	m.OutputTokens += result.OutputTokens
	m.TotalCost += result.Cost
	m.TotalTime += result.Duration
	m.ToolCalls += result.ToolCalls

	if !result.Success {
		m.Errors++
	}
}

// AverageCostPerTurn returns the average cost per turn
func (m *AgentMetrics) AverageCostPerTurn() float64 {
	if m.TotalTurns == 0 {
		return 0.0
	}
	return m.TotalCost / float64(m.TotalTurns)
}

// AverageTokensPerTurn returns the average tokens used per turn
func (m *AgentMetrics) AverageTokensPerTurn() float64 {
	if m.TotalTurns == 0 {
		return 0.0
	}
	return float64(m.TotalTokens) / float64(m.TotalTurns)
}

// AverageDurationPerTurn returns the average duration per turn
func (m *AgentMetrics) AverageDurationPerTurn() time.Duration {
	if m.TotalTurns == 0 {
		return 0
	}
	return m.TotalTime / time.Duration(m.TotalTurns)
}

// SuccessRate returns the percentage of successful turns
func (m *AgentMetrics) SuccessRate() float64 {
	if m.TotalTurns == 0 {
		return 0.0
	}
	successfulTurns := m.TotalTurns - m.Errors
	return (float64(successfulTurns) / float64(m.TotalTurns)) * 100.0
}

// SessionMetrics tracks cumulative metrics for an entire session
type SessionMetrics struct {
	SessionID    string
	TotalTurns   int
	TotalTokens  int
	InputTokens  int
	OutputTokens int
	TotalCost    float64
	TotalTime    time.Duration
	AgentMetrics map[string]*AgentMetrics
}

// NewSessionMetrics creates a new session metrics tracker
func NewSessionMetrics(sessionID string) *SessionMetrics {
	return &SessionMetrics{
		SessionID:    sessionID,
		AgentMetrics: make(map[string]*AgentMetrics),
	}
}

// RecordTurn adds a turn result to the session metrics
func (s *SessionMetrics) RecordTurn(result *TurnResult) {
	// Update session-level metrics
	s.TotalTurns++
	s.TotalTokens += result.TokensUsed
	s.InputTokens += result.InputTokens
	s.OutputTokens += result.OutputTokens
	s.TotalCost += result.Cost
	s.TotalTime += result.Duration

	// Update agent-specific metrics
	if _, exists := s.AgentMetrics[result.AgentID]; !exists {
		s.AgentMetrics[result.AgentID] = &AgentMetrics{
			AgentID: result.AgentID,
		}
	}
	s.AgentMetrics[result.AgentID].UpdateMetrics(result)
}

// GetAgentMetrics returns metrics for a specific agent
func (s *SessionMetrics) GetAgentMetrics(agentID string) *AgentMetrics {
	return s.AgentMetrics[agentID]
}

// CostPerMinute returns the cost rate in dollars per minute
func (s *SessionMetrics) CostPerMinute() float64 {
	if s.TotalTime == 0 {
		return 0.0
	}
	minutes := s.TotalTime.Minutes()
	return s.TotalCost / minutes
}

// TokensPerMinute returns the token usage rate
func (s *SessionMetrics) TokensPerMinute() float64 {
	if s.TotalTime == 0 {
		return 0.0
	}
	minutes := s.TotalTime.Minutes()
	return float64(s.TotalTokens) / minutes
}
