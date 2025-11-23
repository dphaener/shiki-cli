package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name       string
		tokens     int
		model      string
		wantCost   float64
		tolerance  float64
	}{
		{
			name:      "Sonnet 4 - 1M tokens",
			tokens:    1_000_000,
			model:     "claude-sonnet-4",
			wantCost:  9.0, // (3+15)/2 * 1M / 1M = 9.0
			tolerance: 0.01,
		},
		{
			name:      "Sonnet 4.5 - 1M tokens",
			tokens:    1_000_000,
			model:     "claude-sonnet-4-5",
			wantCost:  9.0,
			tolerance: 0.01,
		},
		{
			name:      "Opus 4 - 1M tokens",
			tokens:    1_000_000,
			model:     "claude-opus-4",
			wantCost:  45.0, // (15+75)/2 * 1M / 1M = 45.0
			tolerance: 0.01,
		},
		{
			name:      "Haiku 4 - 1M tokens",
			tokens:    1_000_000,
			model:     "claude-haiku-4",
			wantCost:  0.75, // (0.25+1.25)/2 * 1M / 1M = 0.75
			tolerance: 0.01,
		},
		{
			name:      "Unknown model defaults to Sonnet",
			tokens:    1_000_000,
			model:     "unknown-model",
			wantCost:  9.0,
			tolerance: 0.01,
		},
		{
			name:      "Sonnet 4 - 100K tokens",
			tokens:    100_000,
			model:     "claude-sonnet-4",
			wantCost:  0.9, // 9.0 * 0.1
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCost(tt.tokens, tt.model)
			assert.InDelta(t, tt.wantCost, cost, tt.tolerance)
		})
	}
}

func TestCalculateCostDetailed(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		model        string
		wantCost     float64
		tolerance    float64
	}{
		{
			name:         "Sonnet 4 - balanced input/output",
			inputTokens:  500_000,
			outputTokens: 500_000,
			model:        "claude-sonnet-4",
			wantCost:     9.0, // (500K/1M * 3) + (500K/1M * 15) = 1.5 + 7.5 = 9.0
			tolerance:    0.01,
		},
		{
			name:         "Sonnet 4 - input heavy",
			inputTokens:  900_000,
			outputTokens: 100_000,
			model:        "claude-sonnet-4",
			wantCost:     4.2, // (900K/1M * 3) + (100K/1M * 15) = 2.7 + 1.5 = 4.2
			tolerance:    0.01,
		},
		{
			name:         "Sonnet 4 - output heavy",
			inputTokens:  100_000,
			outputTokens: 900_000,
			model:        "claude-sonnet-4",
			wantCost:     13.8, // (100K/1M * 3) + (900K/1M * 15) = 0.3 + 13.5 = 13.8
			tolerance:    0.01,
		},
		{
			name:         "Opus 4 - balanced",
			inputTokens:  500_000,
			outputTokens: 500_000,
			model:        "claude-opus-4",
			wantCost:     45.0, // (500K/1M * 15) + (500K/1M * 75) = 7.5 + 37.5 = 45.0
			tolerance:    0.01,
		},
		{
			name:         "Haiku 4 - balanced",
			inputTokens:  500_000,
			outputTokens: 500_000,
			model:        "claude-haiku-4",
			wantCost:     0.75, // (500K/1M * 0.25) + (500K/1M * 1.25) = 0.125 + 0.625 = 0.75
			tolerance:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCostDetailed(tt.inputTokens, tt.outputTokens, tt.model)
			assert.InDelta(t, tt.wantCost, cost, tt.tolerance)
		})
	}
}

func TestTurnResult(t *testing.T) {
	result := &TurnResult{
		AgentID:      "test_agent",
		TurnNumber:   1,
		Duration:     time.Second,
		TokensUsed:   1000,
		InputTokens:  600,
		OutputTokens: 400,
		Cost:         0.1,
		Response:     "Test response",
		ToolCalls:    3,
		Success:      true,
		Error:        "",
	}

	assert.Equal(t, "test_agent", result.AgentID)
	assert.Equal(t, 1, result.TurnNumber)
	assert.Equal(t, time.Second, result.Duration)
	assert.Equal(t, 1000, result.TokensUsed)
	assert.Equal(t, 600, result.InputTokens)
	assert.Equal(t, 400, result.OutputTokens)
	assert.InDelta(t, 0.1, result.Cost, 0.001)
	assert.Equal(t, "Test response", result.Response)
	assert.Equal(t, 3, result.ToolCalls)
	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
}

func TestAgentMetrics_UpdateMetrics(t *testing.T) {
	metrics := &AgentMetrics{
		AgentID: "test_agent",
	}

	result1 := &TurnResult{
		AgentID:      "test_agent",
		TurnNumber:   1,
		Duration:     time.Second,
		TokensUsed:   1000,
		InputTokens:  600,
		OutputTokens: 400,
		Cost:         0.1,
		ToolCalls:    2,
		Success:      true,
	}

	result2 := &TurnResult{
		AgentID:      "test_agent",
		TurnNumber:   2,
		Duration:     2 * time.Second,
		TokensUsed:   2000,
		InputTokens:  1200,
		OutputTokens: 800,
		Cost:         0.2,
		ToolCalls:    3,
		Success:      false,
	}

	metrics.UpdateMetrics(result1)
	assert.Equal(t, 1, metrics.TotalTurns)
	assert.Equal(t, 1000, metrics.TotalTokens)
	assert.Equal(t, 600, metrics.InputTokens)
	assert.Equal(t, 400, metrics.OutputTokens)
	assert.InDelta(t, 0.1, metrics.TotalCost, 0.001)
	assert.Equal(t, time.Second, metrics.TotalTime)
	assert.Equal(t, 2, metrics.ToolCalls)
	assert.Equal(t, 0, metrics.Errors)

	metrics.UpdateMetrics(result2)
	assert.Equal(t, 2, metrics.TotalTurns)
	assert.Equal(t, 3000, metrics.TotalTokens)
	assert.Equal(t, 1800, metrics.InputTokens)
	assert.Equal(t, 1200, metrics.OutputTokens)
	assert.InDelta(t, 0.3, metrics.TotalCost, 0.001)
	assert.Equal(t, 3*time.Second, metrics.TotalTime)
	assert.Equal(t, 5, metrics.ToolCalls)
	assert.Equal(t, 1, metrics.Errors)
}

func TestAgentMetrics_AverageMethods(t *testing.T) {
	metrics := &AgentMetrics{
		AgentID:      "test_agent",
		TotalTurns:   4,
		TotalTokens:  4000,
		InputTokens:  2400,
		OutputTokens: 1600,
		TotalCost:    0.4,
		TotalTime:    4 * time.Second,
		ToolCalls:    8,
		Errors:       1,
	}

	assert.InDelta(t, 0.1, metrics.AverageCostPerTurn(), 0.001)
	assert.InDelta(t, 1000.0, metrics.AverageTokensPerTurn(), 0.1)
	assert.Equal(t, time.Second, metrics.AverageDurationPerTurn())
	assert.InDelta(t, 75.0, metrics.SuccessRate(), 0.1) // 3/4 = 75%
}

func TestAgentMetrics_ZeroTurns(t *testing.T) {
	metrics := &AgentMetrics{
		AgentID: "test_agent",
	}

	assert.Equal(t, 0.0, metrics.AverageCostPerTurn())
	assert.Equal(t, 0.0, metrics.AverageTokensPerTurn())
	assert.Equal(t, time.Duration(0), metrics.AverageDurationPerTurn())
	assert.Equal(t, 0.0, metrics.SuccessRate())
}

func TestNewSessionMetrics(t *testing.T) {
	metrics := NewSessionMetrics("test-session")

	assert.Equal(t, "test-session", metrics.SessionID)
	assert.Equal(t, 0, metrics.TotalTurns)
	assert.NotNil(t, metrics.AgentMetrics)
	assert.Empty(t, metrics.AgentMetrics)
}

func TestSessionMetrics_RecordTurn(t *testing.T) {
	metrics := NewSessionMetrics("test-session")

	result1 := &TurnResult{
		AgentID:      "agent1",
		TurnNumber:   1,
		Duration:     time.Second,
		TokensUsed:   1000,
		InputTokens:  600,
		OutputTokens: 400,
		Cost:         0.1,
		ToolCalls:    2,
		Success:      true,
	}

	result2 := &TurnResult{
		AgentID:      "agent2",
		TurnNumber:   2,
		Duration:     2 * time.Second,
		TokensUsed:   2000,
		InputTokens:  1200,
		OutputTokens: 800,
		Cost:         0.2,
		ToolCalls:    3,
		Success:      true,
	}

	metrics.RecordTurn(result1)
	assert.Equal(t, 1, metrics.TotalTurns)
	assert.Equal(t, 1000, metrics.TotalTokens)
	assert.InDelta(t, 0.1, metrics.TotalCost, 0.001)

	agent1Metrics := metrics.GetAgentMetrics("agent1")
	assert.NotNil(t, agent1Metrics)
	assert.Equal(t, 1, agent1Metrics.TotalTurns)
	assert.Equal(t, 1000, agent1Metrics.TotalTokens)

	metrics.RecordTurn(result2)
	assert.Equal(t, 2, metrics.TotalTurns)
	assert.Equal(t, 3000, metrics.TotalTokens)
	assert.InDelta(t, 0.3, metrics.TotalCost, 0.001)

	agent2Metrics := metrics.GetAgentMetrics("agent2")
	assert.NotNil(t, agent2Metrics)
	assert.Equal(t, 1, agent2Metrics.TotalTurns)
	assert.Equal(t, 2000, agent2Metrics.TotalTokens)
}

func TestSessionMetrics_Rates(t *testing.T) {
	metrics := NewSessionMetrics("test-session")

	// Simulate 10 turns over 2 minutes with 10K tokens and $1 cost
	metrics.TotalTurns = 10
	metrics.TotalTokens = 10_000
	metrics.TotalCost = 1.0
	metrics.TotalTime = 2 * time.Minute

	costPerMin := metrics.CostPerMinute()
	assert.InDelta(t, 0.5, costPerMin, 0.001) // $1 / 2min = $0.50/min

	tokensPerMin := metrics.TokensPerMinute()
	assert.InDelta(t, 5000.0, tokensPerMin, 0.1) // 10K / 2min = 5K/min
}

func TestSessionMetrics_ZeroDuration(t *testing.T) {
	metrics := NewSessionMetrics("test-session")

	assert.Equal(t, 0.0, metrics.CostPerMinute())
	assert.Equal(t, 0.0, metrics.TokensPerMinute())
}
