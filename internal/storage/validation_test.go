package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateFeatureName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid name",
			input:     "user-authentication",
			wantError: false,
		},
		{
			name:      "valid name with spaces",
			input:     "User Authentication System",
			wantError: false,
		},
		{
			name:      "valid name with numbers",
			input:     "API v2 Integration",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
			errorMsg:  "feature name cannot be empty",
		},
		{
			name:      "whitespace only",
			input:     "   \t\n  ",
			wantError: true,
			errorMsg:  "feature name cannot be whitespace-only",
		},
		{
			name:      "too long",
			input:     "this-is-a-very-long-feature-name-that-exceeds-one-hundred-characters-and-should-definitely-be-rejected-due-to-length",
			wantError: true,
			errorMsg:  "feature name cannot exceed 100 characters",
		},
		{
			name:      "directory traversal with dots",
			input:     "../malicious-feature",
			wantError: true,
			errorMsg:  "feature name cannot contain directory traversal characters",
		},
		{
			name:      "directory traversal with slash",
			input:     "feature/with/slash",
			wantError: true,
			errorMsg:  "feature name cannot contain directory traversal characters",
		},
		{
			name:      "directory traversal with backslash",
			input:     "feature\\with\\backslash",
			wantError: true,
			errorMsg:  "feature name cannot contain directory traversal characters",
		},
		{
			name:      "null character",
			input:     "feature\x00name",
			wantError: true,
			errorMsg:  "feature name contains invalid control characters",
		},
		{
			name:      "newline character",
			input:     "feature\nname",
			wantError: true,
			errorMsg:  "feature name contains invalid control characters",
		},
		{
			name:      "tab character",
			input:     "feature\tname",
			wantError: true,
			errorMsg:  "feature name contains invalid control characters",
		},
		{
			name:      "carriage return",
			input:     "feature\rname",
			wantError: true,
			errorMsg:  "feature name contains invalid control characters",
		},
		{
			name:      "unicode characters",
			input:     "功能名称",
			wantError: false,
		},
		{
			name:      "mixed unicode and ascii",
			input:     "User Authentication 用户认证",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureName(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateSlugFromName(t *testing.T) {
	tests := []struct {
		name           string
		featureName    string
		featureNumber  int
		expectedSlug   string
	}{
		{
			name:          "simple name",
			featureName:   "user-authentication",
			featureNumber: 1,
			expectedSlug:  "001-user-authentication",
		},
		{
			name:          "name with spaces",
			featureName:   "User Authentication System",
			featureNumber: 42,
			expectedSlug:  "042-user-authentication-system",
		},
		{
			name:          "name with special characters",
			featureName:   "API v2.0 Integration!",
			featureNumber: 5,
			expectedSlug:  "005-api-v2-0-integration",
		},
		{
			name:          "name with multiple spaces",
			featureName:   "Real    Time   Collaboration",
			featureNumber: 10,
			expectedSlug:  "010-real-time-collaboration",
		},
		{
			name:          "unicode characters",
			featureName:   "功能名称",
			featureNumber: 7,
			expectedSlug:  "007-功能名称",
		},
		{
			name:          "mixed unicode and ascii",
			featureName:   "User Authentication 用户认证",
			featureNumber: 3,
			expectedSlug:  "003-user-authentication-用户认证",
		},
		{
			name:          "empty after sanitization",
			featureName:   "!!!@@@###",
			featureNumber: 8,
			expectedSlug:  "008-feature",
		},
		{
			name:          "very long name gets truncated",
			featureName:   "this-is-a-very-long-feature-name-that-should-be-truncated",
			featureNumber: 15,
			expectedSlug:  "015-this-is-a-very-long-feature-name-that-should-be",
		},
		{
			name:          "truncation at word boundary",
			featureName:   "this-is-a-very-long-feature-name-that-should-be-truncated-at-word-boundary",
			featureNumber: 20,
			expectedSlug:  "020-this-is-a-very-long-feature-name-that-should-be",
		},
		{
			name:          "name with leading/trailing spaces",
			featureName:   "   user authentication   ",
			featureNumber: 12,
			expectedSlug:  "012-user-authentication",
		},
		{
			name:          "name with punctuation",
			featureName:   "User's Authentication & Authorization",
			featureNumber: 25,
			expectedSlug:  "025-user-s-authentication-authorization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateSlugFromName(tt.featureName, tt.featureNumber)
			assert.Equal(t, tt.expectedSlug, result)
		})
	}
}

func TestCreateSlugBackwardCompatibility(t *testing.T) {
	// Test that CreateSlug still works and calls CreateSlugFromName
	name := "User Authentication"
	number := 5

	oldResult := CreateSlug(name, number)
	newResult := CreateSlugFromName(name, number)

	assert.Equal(t, newResult, oldResult, "CreateSlug should maintain backward compatibility")
	assert.Equal(t, "005-user-authentication", oldResult)
}

// Note: PromptForFeatureName() is difficult to unit test as it relies on os.Stdin
// In a real implementation, we would refactor it to accept an io.Reader for testability
// For now, the function is implemented and ready for integration testing