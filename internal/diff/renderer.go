package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
)

// Renderer handles the visual formatting of diffs for terminal display
type Renderer struct {
	maxLines  int
	maxWidth  int
	showLines bool
}

// NewRenderer creates a new diff renderer with default settings
func NewRenderer() *Renderer {
	return &Renderer{
		maxLines:  100,
		maxWidth:  80,
		showLines: true,
	}
}

// SetMaxLines configures the maximum number of diff lines to display
func (r *Renderer) SetMaxLines(maxLines int) {
	if maxLines > 0 {
		r.maxLines = maxLines
	}
}

// SetMaxWidth configures the maximum width for line wrapping
func (r *Renderer) SetMaxWidth(maxWidth int) {
	if maxWidth > 0 {
		r.maxWidth = maxWidth
	}
}

// SetShowLines configures whether to show line numbers
func (r *Renderer) SetShowLines(showLines bool) {
	r.showLines = showLines
}

// FormatDiffForTerminal renders a unified diff with colors for terminal display
func (r *Renderer) FormatDiffForTerminal(fileDiff *FileDiff, width int) string {
	if fileDiff == nil {
		return ""
	}

	// Handle binary files
	if fileDiff.IsBinary {
		return r.formatBinaryFileDiff(fileDiff)
	}

	// Handle empty diffs (no changes)
	if fileDiff.UnifiedDiff == "" {
		return ""
	}

	var result strings.Builder

	// Add file header
	header := r.createDiffHeader(fileDiff.FilePath)
	result.WriteString(header)
	result.WriteString("\n")

	// Process diff lines
	diffLines := strings.Split(fileDiff.UnifiedDiff, "\n")
	formattedLines := r.formatDiffLines(diffLines, width)

	result.WriteString(formattedLines)

	return result.String()
}

// createDiffHeader creates a styled header showing the file path
func (r *Renderer) createDiffHeader(filePath string) string {
	// Sanitize file path for display
	displayPath := SanitizeFilePath(filePath)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Info).
		Bold(true)

	return headerStyle.Render(fmt.Sprintf("📝 %s", displayPath))
}

// formatBinaryFileDiff creates a display for binary file modifications
func (r *Renderer) formatBinaryFileDiff(fileDiff *FileDiff) string {
	binaryStyle := lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Italic(true)

	iconStyle := lipgloss.NewStyle().
		Foreground(theme.Warning)

	icon := iconStyle.Render("📦")
	message := binaryStyle.Render(fileDiff.UnifiedDiff)

	return fmt.Sprintf("%s %s", icon, message)
}

// formatDiffLines applies color formatting to individual diff lines
func (r *Renderer) formatDiffLines(lines []string, width int) string {
	var result strings.Builder
	displayedLines := 0

	for _, line := range lines {
		// Skip empty lines at the start (diff headers we don't need)
		if line == "" && displayedLines == 0 {
			continue
		}

		// Check if we've hit the display limit
		if displayedLines >= r.maxLines {
			omittedCount := len(lines) - displayedLines
			if omittedCount > 0 {
				omissionStyle := lipgloss.NewStyle().
					Foreground(theme.TextMuted).
					Italic(true)
				result.WriteString(omissionStyle.Render(fmt.Sprintf("... %d lines omitted ...", omittedCount)))
				result.WriteString("\n")
			}
			break
		}

		// Format line based on diff markers
		formattedLine := r.formatSingleLine(line, width)
		if formattedLine != "" {
			result.WriteString(formattedLine)
			result.WriteString("\n")
			displayedLines++
		}
	}

	return result.String()
}

// formatSingleLine applies appropriate styling to a single diff line
func (r *Renderer) formatSingleLine(line string, width int) string {
	if line == "" {
		return ""
	}

	// Skip diff file headers (---, +++, @@@ markers handled separately)
	if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
		return ""
	}

	// Handle range headers (@@)
	if strings.HasPrefix(line, "@@") {
		style := lipgloss.NewStyle().
			Foreground(theme.Info).
			Faint(true)
		return "    " + style.Render(line)
	}

	// Handle addition lines (+)
	if strings.HasPrefix(line, "+") {
		style := lipgloss.NewStyle().
			Foreground(theme.Success)
		content := line[1:] // Remove the + marker
		return "    " + style.Render("+") + " " + content
	}

	// Handle deletion lines (-)
	if strings.HasPrefix(line, "-") {
		style := lipgloss.NewStyle().
			Foreground(theme.Error)
		content := line[1:] // Remove the - marker
		return "    " + style.Render("-") + " " + content
	}

	// Handle context lines (space or no marker)
	contextStyle := lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	// Context lines should have a space prefix in unified diff
	if strings.HasPrefix(line, " ") {
		content := line[1:] // Remove the space marker
		return "    " + contextStyle.Render(" ") + " " + content
	}

	// Handle lines without proper markers (shouldn't happen in well-formed diffs)
	return "    " + contextStyle.Render("  " + line)
}

// ApplyLineWrapping wraps long lines to fit within terminal width
func (r *Renderer) ApplyLineWrapping(content string, width int) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= width {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Wrap long lines
		prefix := "    "
		if strings.HasPrefix(line, "    ") {
			prefix = strings.Repeat(" ", 8) // Extra indent for wrapped lines
			line = line[4:]                 // Remove original prefix
		}

		maxContentWidth := width - len(prefix)
		if maxContentWidth <= 0 {
			maxContentWidth = 40 // Minimum reasonable width
		}

		for len(line) > maxContentWidth {
			breakPoint := maxContentWidth
			// Try to break at a word boundary
			spaceIndex := strings.LastIndex(line[:maxContentWidth], " ")
			if spaceIndex > maxContentWidth/2 {
				breakPoint = spaceIndex
			}

			wrappedLines = append(wrappedLines, prefix+line[:breakPoint])
			line = line[breakPoint:]
			if strings.HasPrefix(line, " ") {
				line = line[1:] // Remove leading space on continuation
			}
		}

		if len(line) > 0 {
			wrappedLines = append(wrappedLines, prefix+line)
		}
	}

	return strings.Join(wrappedLines, "\n")
}

// CreateDiffSummary creates a brief summary for large diffs
func CreateDiffSummary(fileDiff *FileDiff) string {
	if fileDiff == nil {
		return ""
	}

	summaryStyle := lipgloss.NewStyle().
		Foreground(theme.Info).
		Italic(true)

	if fileDiff.IsBinary {
		return summaryStyle.Render("Binary file modified")
	}

	if fileDiff.IsLarge {
		return summaryStyle.Render(fmt.Sprintf("Large diff (%d total lines)", fileDiff.TotalLines))
	}

	// Count additions and deletions
	lines := strings.Split(fileDiff.UnifiedDiff, "\n")
	additions := 0
	deletions := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}

	if additions > 0 || deletions > 0 {
		return summaryStyle.Render(fmt.Sprintf("+%d -%d", additions, deletions))
	}

	return ""
}