package components

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownRenderer provides thread-safe markdown rendering with proper width handling
type MarkdownRenderer struct {
	mu        sync.Mutex
	renderers map[int]*glamour.TermRenderer // Cache renderers by width
}

var (
	sharedRenderer     *MarkdownRenderer
	sharedRendererOnce sync.Once
)

// GetMarkdownRenderer returns the shared markdown renderer instance
func GetMarkdownRenderer() *MarkdownRenderer {
	sharedRendererOnce.Do(func() {
		sharedRenderer = &MarkdownRenderer{
			renderers: make(map[int]*glamour.TermRenderer),
		}
	})
	return sharedRenderer
}

// Render renders markdown text with the specified width
// Returns the rendered text and any error encountered
func (r *MarkdownRenderer) Render(text string, width int) (string, error) {
	if text == "" {
		return "", nil
	}

	// Normalize width to avoid cache bloat (round to nearest 10)
	normalizedWidth := (width / 10) * 10
	if normalizedWidth < 40 {
		normalizedWidth = 40 // Minimum width
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Get or create renderer for this width
	renderer, exists := r.renderers[normalizedWidth]
	if !exists {
		var err error
		renderer, err = glamour.NewTermRenderer(
			glamour.WithStyles(sekkeiStyle()),
			glamour.WithWordWrap(normalizedWidth),
		)
		if err != nil {
			return "", err
		}
		r.renderers[normalizedWidth] = renderer

		// Limit cache size to prevent memory bloat
		if len(r.renderers) > 10 {
			// Clear cache and keep only current renderer
			r.renderers = map[int]*glamour.TermRenderer{
				normalizedWidth: renderer,
			}
		}
	}

	// Render the markdown
	rendered, err := renderer.Render(text)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(rendered), nil
}

// RenderWithStyle renders markdown and applies a lipgloss style
func (r *MarkdownRenderer) RenderWithStyle(text string, width int, style lipgloss.Style) string {
	rendered, err := r.Render(text, width)
	if err != nil {
		// Fallback to plain text with simple word wrapping
		return style.Render(wordWrap(text, width))
	}
	return style.Render(rendered)
}

// wordWrap provides simple word wrapping fallback
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// sekkeiStyle returns a custom glamour style using the Sekkei Design System colors
func sekkeiStyle() ansi.StyleConfig {
	// Emerald color for headings (matches theme.Primary #10b981)
	emerald := "#10b981"
	emeraldDark := "#059669"
	sky := "#0ea5e9"
	slate400 := "#94a3b8"
	slate200 := "#e2e8f0"

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "",
				BlockSuffix: "",
			},
			Margin: uintPtr(0),
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(emerald),
				Bold:  boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(emerald),
				Bold:   boolPtr(true),
				Prefix: "# ",
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(emerald),
				Bold:   boolPtr(true),
				Prefix: "## ",
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(emeraldDark),
				Bold:   boolPtr(true),
				Prefix: "### ",
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(emeraldDark),
				Bold:   boolPtr(true),
				Prefix: "#### ",
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(emeraldDark),
				Bold:   boolPtr(true),
				Prefix: "##### ",
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(emeraldDark),
				Bold:   boolPtr(true),
				Prefix: "###### ",
			},
		},
		Strong: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Link: ansi.StylePrimitive{
			Color:     stringPtr(sky),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: stringPtr(sky),
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(slate200),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: uintPtr(1),
			},
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{},
			LevelIndent: 2,
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		// Enumeration handles ordered list item formatting
		// The markdown parser handles numbering, this provides the period and space
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Paragraph: ansi.StyleBlock{},
		Text: ansi.StylePrimitive{},
		HorizontalRule: ansi.StylePrimitive{
			Color:  stringPtr(slate400),
			Format: "───────────────────",
		},
	}
}

// Helper functions for pointer values
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func uintPtr(u uint) *uint       { return &u }
