package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureNamingValidation(t *testing.T) {
	tests := []struct {
		name        string
		featureName string
		wantError   bool
	}{
		{
			name:        "valid simple name",
			featureName: "user-authentication",
			wantError:   false,
		},
		{
			name:        "valid name with spaces",
			featureName: "Real Time Notifications",
			wantError:   false,
		},
		{
			name:        "valid name with Unicode",
			featureName: "用户认证",
			wantError:   false,
		},
		{
			name:        "invalid name with directory traversal",
			featureName: "../malicious-name",
			wantError:   true,
		},
		{
			name:        "invalid empty name",
			featureName: "",
			wantError:   true,
		},
		{
			name:        "invalid whitespace-only name",
			featureName: "   \t\n  ",
			wantError:   true,
		},
		{
			name:        "invalid very long name",
			featureName: "this-is-an-extremely-long-feature-name-that-exceeds-the-maximum-allowed-length-of-one-hundred-characters-and-should-be-rejected",
			wantError:   true,
		},
		{
			name:        "invalid name with null character",
			featureName: "feature\x00name",
			wantError:   true,
		},
		{
			name:        "invalid name with backslash",
			featureName: "feature\\name",
			wantError:   true,
		},
		{
			name:        "invalid name with forward slash",
			featureName: "feature/name",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureName(tt.featureName)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSlugGenerationFromNames(t *testing.T) {
	tests := []struct {
		name         string
		featureName  string
		featureNum   int
		expectedSlug string
	}{
		{
			name:         "simple name",
			featureName:  "user-authentication",
			featureNum:   1,
			expectedSlug: "001-user-authentication",
		},
		{
			name:         "name with spaces",
			featureName:  "Real Time Notifications",
			featureNum:   42,
			expectedSlug: "042-real-time-notifications",
		},
		{
			name:         "name with special characters",
			featureName:  "API v2.0 Integration!",
			featureNum:   5,
			expectedSlug: "005-api-v2-0-integration",
		},
		{
			name:         "Unicode characters",
			featureName:  "用户认证",
			featureNum:   7,
			expectedSlug: "007-用户认证",
		},
		{
			name:         "only special characters",
			featureName:  "!!!@@@###",
			featureNum:   8,
			expectedSlug: "008-feature",
		},
		{
			name:         "very long name gets truncated",
			featureName:  "this-is-a-very-long-feature-name-that-should-get-truncated-because-it-exceeds-the-maximum-slug-length",
			featureNum:   15,
			expectedSlug: "015-this-is-a-very-long-feature-name-that-should-get",
		},
		{
			name:         "leading and trailing spaces",
			featureName:  "   feature name   ",
			featureNum:   20,
			expectedSlug: "020-feature-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug := CreateSlugFromName(tt.featureName, tt.featureNum)
			assert.Equal(t, tt.expectedSlug, slug)
		})
	}
}

func TestFeatureSetupConfigStructure(t *testing.T) {
	// Test that the new FeatureSetupConfig structure works correctly
	config := FeatureSetupConfig{
		FeatureName: "test-feature",
	}

	// Validate that we can validate the feature name
	err := ValidateFeatureName(config.FeatureName)
	require.NoError(t, err)

	// Validate that we can create a slug from it
	slug := CreateSlugFromName(config.FeatureName, 1)
	assert.Equal(t, "001-test-feature", slug)
}