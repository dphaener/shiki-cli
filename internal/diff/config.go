package diff

// Config represents configuration options for diff generation and display
type Config struct {
	// Display limits
	MaxDiffLines    int  `json:"max_diff_lines"`    // Maximum lines to display in a diff (default: 100)
	ContextLines    int  `json:"context_lines"`     // Context lines around changes (default: 3)
	MaxFileSize     int64 `json:"max_file_size"`     // Maximum file size to process in bytes (default: 1MB)
	ShowLineNumbers bool  `json:"show_line_numbers"` // Whether to show line numbers (default: true)
	ColorEnabled    bool  `json:"color_enabled"`     // Whether to use colors in output (default: true)

	// Performance limits
	MaxProcessingTime int `json:"max_processing_time_ms"` // Max processing time in milliseconds (default: 2000)

	// Feature toggles
	Enabled         bool `json:"enabled"`          // Whether diff display is enabled (default: true)
	BinaryDetection bool `json:"binary_detection"` // Whether to detect binary files (default: true)
	LargeDiffSummary bool `json:"large_diff_summary"` // Whether to show summary for large diffs (default: true)

	// Terminal display options
	MaxTerminalWidth int `json:"max_terminal_width"` // Maximum terminal width to consider (default: 120)
	WrapLongLines   bool `json:"wrap_long_lines"`   // Whether to wrap long lines (default: true)
	IndentSize      int  `json:"indent_size"`       // Indentation size for diff content (default: 4)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		// Display limits
		MaxDiffLines:    100,
		ContextLines:    3,
		MaxFileSize:     1024 * 1024, // 1MB
		ShowLineNumbers: true,
		ColorEnabled:    true,

		// Performance limits
		MaxProcessingTime: 2000, // 2 seconds

		// Feature toggles
		Enabled:          true,
		BinaryDetection:  true,
		LargeDiffSummary: true,

		// Terminal display options
		MaxTerminalWidth: 120,
		WrapLongLines:    true,
		IndentSize:       4,
	}
}

// ApplyToGenerator applies configuration to a Generator
func (c *Config) ApplyToGenerator(generator *Generator) {
	generator.SetMaxLines(c.MaxDiffLines)
	generator.SetContextLines(c.ContextLines)
	generator.maxFileSize = c.MaxFileSize
}

// ApplyToRenderer applies configuration to a Renderer
func (c *Config) ApplyToRenderer(renderer *Renderer) {
	renderer.SetMaxLines(c.MaxDiffLines)
	renderer.SetMaxWidth(c.MaxTerminalWidth)
	renderer.SetShowLines(c.ShowLineNumbers)
}

// Validate checks if the configuration values are valid and sets defaults for invalid values
func (c *Config) Validate() {
	if c.MaxDiffLines <= 0 {
		c.MaxDiffLines = 100
	}
	if c.ContextLines < 0 {
		c.ContextLines = 3
	}
	if c.MaxFileSize <= 0 {
		c.MaxFileSize = 1024 * 1024
	}
	if c.MaxProcessingTime <= 0 {
		c.MaxProcessingTime = 2000
	}
	if c.MaxTerminalWidth <= 20 {
		c.MaxTerminalWidth = 120
	}
	if c.IndentSize < 0 {
		c.IndentSize = 4
	}

	// Reasonable limits to prevent excessive resource usage
	if c.MaxDiffLines > 1000 {
		c.MaxDiffLines = 1000
	}
	if c.ContextLines > 50 {
		c.ContextLines = 50
	}
	if c.MaxFileSize > 10*1024*1024 { // 10MB
		c.MaxFileSize = 10 * 1024 * 1024
	}
	if c.MaxProcessingTime > 30000 { // 30 seconds
		c.MaxProcessingTime = 30000
	}
	if c.MaxTerminalWidth > 300 {
		c.MaxTerminalWidth = 300
	}
}

// ConfigurableGenerator is a Generator that can be configured
type ConfigurableGenerator struct {
	*Generator
	config *Config
}

// NewConfigurableGenerator creates a new generator with configuration
func NewConfigurableGenerator(config *Config) *ConfigurableGenerator {
	if config == nil {
		config = DefaultConfig()
	}
	config.Validate()

	generator := NewGenerator()
	config.ApplyToGenerator(generator)

	return &ConfigurableGenerator{
		Generator: generator,
		config:    config,
	}
}

// GetConfig returns the current configuration
func (cg *ConfigurableGenerator) GetConfig() *Config {
	return cg.config
}

// UpdateConfig updates the generator configuration
func (cg *ConfigurableGenerator) UpdateConfig(config *Config) {
	if config == nil {
		return
	}
	config.Validate()
	cg.config = config
	config.ApplyToGenerator(cg.Generator)
}

// ConfigurableRenderer is a Renderer that can be configured
type ConfigurableRenderer struct {
	*Renderer
	config *Config
}

// NewConfigurableRenderer creates a new renderer with configuration
func NewConfigurableRenderer(config *Config) *ConfigurableRenderer {
	if config == nil {
		config = DefaultConfig()
	}
	config.Validate()

	renderer := NewRenderer()
	config.ApplyToRenderer(renderer)

	return &ConfigurableRenderer{
		Renderer: renderer,
		config:   config,
	}
}

// GetConfig returns the current configuration
func (cr *ConfigurableRenderer) GetConfig() *Config {
	return cr.config
}

// UpdateConfig updates the renderer configuration
func (cr *ConfigurableRenderer) UpdateConfig(config *Config) {
	if config == nil {
		return
	}
	config.Validate()
	cr.config = config
	config.ApplyToRenderer(cr.Renderer)
}