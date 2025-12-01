package diff

import (
	"encoding/json"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	if config.MaxDiffLines != 100 {
		t.Errorf("Expected MaxDiffLines to be 100, got %d", config.MaxDiffLines)
	}
	if config.ContextLines != 3 {
		t.Errorf("Expected ContextLines to be 3, got %d", config.ContextLines)
	}
	if config.MaxFileSize != 1024*1024 {
		t.Errorf("Expected MaxFileSize to be 1MB, got %d", config.MaxFileSize)
	}
	if !config.Enabled {
		t.Error("Expected diff to be enabled by default")
	}
	if !config.ColorEnabled {
		t.Error("Expected colors to be enabled by default")
	}
	if !config.BinaryDetection {
		t.Error("Expected binary detection to be enabled by default")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected func(*Config) bool
	}{
		{
			name: "negative max lines",
			config: &Config{
				MaxDiffLines: -10,
			},
			expected: func(c *Config) bool {
				return c.MaxDiffLines == 100 // Should be set to default
			},
		},
		{
			name: "excessive max lines",
			config: &Config{
				MaxDiffLines: 2000,
			},
			expected: func(c *Config) bool {
				return c.MaxDiffLines == 1000 // Should be capped
			},
		},
		{
			name: "negative context lines",
			config: &Config{
				ContextLines: -5,
			},
			expected: func(c *Config) bool {
				return c.ContextLines == 3 // Should be set to default
			},
		},
		{
			name: "excessive file size",
			config: &Config{
				MaxFileSize: 50 * 1024 * 1024, // 50MB
			},
			expected: func(c *Config) bool {
				return c.MaxFileSize == 10*1024*1024 // Should be capped to 10MB
			},
		},
		{
			name: "small terminal width",
			config: &Config{
				MaxTerminalWidth: 10,
			},
			expected: func(c *Config) bool {
				return c.MaxTerminalWidth == 120 // Should be set to default
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.config.Validate()
			if !test.expected(test.config) {
				t.Error("Config validation failed")
			}
		})
	}
}

func TestConfigurableGenerator(t *testing.T) {
	config := &Config{
		MaxDiffLines: 50,
		ContextLines: 5,
		MaxFileSize:  512 * 1024,
	}

	generator := NewConfigurableGenerator(config)

	// Test that configuration is applied
	if generator.maxLines != 50 {
		t.Errorf("Expected maxLines to be 50, got %d", generator.maxLines)
	}
	if generator.contextLines != 5 {
		t.Errorf("Expected contextLines to be 5, got %d", generator.contextLines)
	}
	if generator.maxFileSize != 512*1024 {
		t.Errorf("Expected maxFileSize to be 512KB, got %d", generator.maxFileSize)
	}

	// Test configuration update
	newConfig := &Config{
		MaxDiffLines: 75,
		ContextLines: 2,
		MaxFileSize:  256 * 1024,
	}

	generator.UpdateConfig(newConfig)

	if generator.maxLines != 75 {
		t.Errorf("Expected maxLines to be updated to 75, got %d", generator.maxLines)
	}
	if generator.contextLines != 2 {
		t.Errorf("Expected contextLines to be updated to 2, got %d", generator.contextLines)
	}
}

func TestConfigurableRenderer(t *testing.T) {
	config := &Config{
		MaxDiffLines:     80,
		MaxTerminalWidth: 100,
		ShowLineNumbers:  false,
	}

	renderer := NewConfigurableRenderer(config)

	// Test that configuration is applied
	if renderer.maxLines != 80 {
		t.Errorf("Expected maxLines to be 80, got %d", renderer.maxLines)
	}
	if renderer.maxWidth != 100 {
		t.Errorf("Expected maxWidth to be 100, got %d", renderer.maxWidth)
	}
	if renderer.showLines {
		t.Error("Expected showLines to be false")
	}
}

func TestConfigJSONSerialization(t *testing.T) {
	config := DefaultConfig()

	// Test marshaling
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Test unmarshaling
	var restored Config
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify key fields are preserved
	if restored.MaxDiffLines != config.MaxDiffLines {
		t.Error("MaxDiffLines not preserved in JSON serialization")
	}
	if restored.ContextLines != config.ContextLines {
		t.Error("ContextLines not preserved in JSON serialization")
	}
	if restored.Enabled != config.Enabled {
		t.Error("Enabled not preserved in JSON serialization")
	}
	if restored.ColorEnabled != config.ColorEnabled {
		t.Error("ColorEnabled not preserved in JSON serialization")
	}
}

func TestConfigWithNilValues(t *testing.T) {
	// Test that nil config results in default config
	generator := NewConfigurableGenerator(nil)
	defaultConfig := DefaultConfig()

	if generator.maxLines != defaultConfig.MaxDiffLines {
		t.Error("Nil config should result in default values")
	}

	renderer := NewConfigurableRenderer(nil)
	if renderer.maxLines != defaultConfig.MaxDiffLines {
		t.Error("Nil config should result in default values")
	}

	// Test updating with nil config (should be ignored)
	originalMaxLines := generator.maxLines
	generator.UpdateConfig(nil)
	if generator.maxLines != originalMaxLines {
		t.Error("Updating with nil config should not change values")
	}
}

func TestConfigLimits(t *testing.T) {
	config := &Config{
		MaxDiffLines:      10000,  // Too large
		ContextLines:      100,    // Too large
		MaxFileSize:       100*1024*1024, // 100MB, too large
		MaxProcessingTime: 60000,  // 60 seconds, too large
		MaxTerminalWidth:  500,    // Too large
	}

	config.Validate()

	// Check that values are capped to reasonable limits
	if config.MaxDiffLines > 1000 {
		t.Error("MaxDiffLines should be capped to 1000")
	}
	if config.ContextLines > 50 {
		t.Error("ContextLines should be capped to 50")
	}
	if config.MaxFileSize > 10*1024*1024 {
		t.Error("MaxFileSize should be capped to 10MB")
	}
	if config.MaxProcessingTime > 30000 {
		t.Error("MaxProcessingTime should be capped to 30 seconds")
	}
	if config.MaxTerminalWidth > 300 {
		t.Error("MaxTerminalWidth should be capped to 300")
	}
}