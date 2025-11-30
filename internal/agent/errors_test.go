package agent

import (
	"errors"
	"testing"
)

func TestIsBrokenPipeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "failed to write to stdin",
			err:      errors.New("failed to write to stdin: some details"),
			expected: true,
		},
		{
			name:     "file already closed",
			err:      errors.New("write |1: file already closed"),
			expected: true,
		},
		{
			name:     "broken pipe",
			err:      errors.New("write: broken pipe"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "wrapped broken pipe error",
			err:      errors.New("failed to send query: failed to write to stdin: write |1: file already closed"),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("context deadline exceeded"),
			expected: false,
		},
		{
			name:     "other unrelated error",
			err:      errors.New("network timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBrokenPipeError(tt.err)
			if result != tt.expected {
				t.Errorf("IsBrokenPipeError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "broken pipe - recoverable",
			err:      errors.New("failed to write to stdin: file already closed"),
			expected: true,
		},
		{
			name:     "context deadline exceeded - recoverable",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "wrapped timeout",
			err:      errors.New("query failed: context deadline exceeded"),
			expected: true,
		},
		{
			name:     "network timeout - not recoverable by restart",
			err:      errors.New("network timeout"),
			expected: false,
		},
		{
			name:     "permission denied - not recoverable",
			err:      errors.New("permission denied"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRecoverableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRecoverableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
