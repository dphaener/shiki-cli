package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		commit       string
		buildDate    string
		wantContains []string
		wantExitCode int
	}{
		{
			name:      "version with all build info",
			version:   "1.0.0",
			commit:    "abc123def456",
			buildDate: "2025-01-01T00:00:00Z",
			wantContains: []string{
				"shiki version 1.0.0",
				"commit:     abc123def456",
				"built:      2025-01-01T00:00:00Z",
			},
			wantExitCode: ExitSuccess,
		},
		{
			name:      "version with dev build",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
			wantContains: []string{
				"shiki version dev",
				"commit:     unknown",
				"built:      unknown",
			},
			wantExitCode: ExitSuccess,
		},
		{
			name:      "version with short commit",
			version:   "0.1.0",
			commit:    "abc123",
			buildDate: "2025-11-23",
			wantContains: []string{
				"shiki version 0.1.0",
				"commit:     abc123",
				"built:      2025-11-23",
			},
			wantExitCode: ExitSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			cmd := NewVersionCommand(tt.version, tt.commit, tt.buildDate)
			cmd.SetOut(&stdout)

			err := cmd.Execute()
			require.NoError(t, err)

			output := stdout.String()

			// Check output contains expected substrings
			for _, substr := range tt.wantContains {
				assert.Contains(t, output, substr)
			}
		})
	}
}

func TestVersionCommand_GoVersion(t *testing.T) {
	var stdout bytes.Buffer

	cmd := NewVersionCommand("1.0.0", "abc123", "2025-01-01")
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()

	// Should contain Go version
	assert.Contains(t, output, "go version:")
}
