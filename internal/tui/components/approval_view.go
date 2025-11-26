package components

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)

// ApprovalView displays a full-screen artifact preview with approval overlay
type ApprovalView struct {
	phase        types.WorkflowPhase
	artifactPath string
	artifactName string
	content      string
	width        int
	height       int
	scrollOffset int
}

// NewApprovalView creates a new approval view
func NewApprovalView(phase types.WorkflowPhase, artifactPath string) *ApprovalView {
	av := &ApprovalView{
		phase:        phase,
		artifactPath: artifactPath,
		scrollOffset: 0,
	}

	// Set artifact name based on phase
	switch phase {
	case types.WorkflowPhaseApproveSpec:
		av.artifactName = "Specification"
	case types.WorkflowPhaseApprovePlan:
		av.artifactName = "Implementation Plan"
	case types.WorkflowPhaseApproveTasks:
		av.artifactName = "Task Breakdown"
	default:
		av.artifactName = "Artifact"
	}

	// Load content
	av.loadContent()

	return av
}

// loadContent loads the artifact content from disk
func (av *ApprovalView) loadContent() {
	if av.artifactPath == "" {
		av.content = "(No artifact file specified)"
		return
	}

	data, err := os.ReadFile(av.artifactPath)
	if err != nil {
		av.content = fmt.Sprintf("(Could not load artifact: %v)", err)
		return
	}

	av.content = string(data)
}

// SetSize sets the view dimensions
func (av *ApprovalView) SetSize(width, height int) {
	av.width = width
	av.height = height
}

// ScrollUp scrolls the content up
func (av *ApprovalView) ScrollUp(lines int) {
	av.scrollOffset -= lines
	if av.scrollOffset < 0 {
		av.scrollOffset = 0
	}
}

// ScrollDown scrolls the content down
func (av *ApprovalView) ScrollDown(lines int) {
	av.scrollOffset += lines
	// Will be clamped in View()
}

// View renders the approval view
func (av ApprovalView) View() string {
	if av.width == 0 || av.height == 0 {
		return "Initializing..."
	}

	// Calculate dimensions
	overlayHeight := 7
	contentHeight := av.height - overlayHeight - 4 // padding

	// Render the artifact content
	contentView := av.renderContent(contentHeight)

	// Render the approval overlay
	overlayView := av.renderOverlay()

	// Combine with the overlay at the bottom
	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentView,
		overlayView,
	)
}

// renderContent renders the scrollable artifact content
func (av *ApprovalView) renderContent(height int) string {
	// Split content into lines
	lines := strings.Split(av.content, "\n")

	// Clamp scroll offset
	maxScroll := len(lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if av.scrollOffset > maxScroll {
		av.scrollOffset = maxScroll
	}

	// Get visible lines
	endLine := av.scrollOffset + height
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleLines := lines[av.scrollOffset:endLine]

	// Render each line with basic markdown styling
	var renderedLines []string
	for _, line := range visibleLines {
		renderedLines = append(renderedLines, av.renderMarkdownLine(line))
	}

	// Pad to fill height
	for len(renderedLines) < height {
		renderedLines = append(renderedLines, "")
	}

	content := strings.Join(renderedLines, "\n")

	// Add scroll indicators
	scrollInfo := ""
	if len(lines) > height {
		scrollInfo = fmt.Sprintf(" [%d-%d of %d lines]", av.scrollOffset+1, endLine, len(lines))
	}

	header := approvalContentHeaderStyle.
		Width(av.width - 2).
		Render(fmt.Sprintf("📄 %s%s", av.artifactName, scrollInfo))

	return approvalContentStyle.
		Width(av.width).
		Height(height + 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, content))
}

// renderMarkdownLine applies basic markdown styling to a line
func (av *ApprovalView) renderMarkdownLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Headers
	if strings.HasPrefix(trimmed, "# ") {
		return approvalH1Style.Render(trimmed)
	}
	if strings.HasPrefix(trimmed, "## ") {
		return approvalH2Style.Render(trimmed)
	}
	if strings.HasPrefix(trimmed, "### ") {
		return approvalH3Style.Render(trimmed)
	}

	// Checkboxes
	if strings.HasPrefix(trimmed, "- [ ]") {
		return approvalCheckboxStyle.Render("☐") + " " + strings.TrimPrefix(trimmed, "- [ ]")
	}
	if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
		checkbox := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x]"), "- [X]")
		return approvalCheckboxDoneStyle.Render("☑") + " " + approvalStrikethroughStyle.Render(checkbox)
	}

	// Bullet points
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return approvalBulletStyle.Render("•") + " " + trimmed[2:]
	}

	// Code blocks
	if strings.HasPrefix(trimmed, "```") {
		return approvalCodeBlockStyle.Render(trimmed)
	}

	// Frontmatter delimiter
	if trimmed == "---" {
		return approvalDividerStyle.Render(strings.Repeat("─", av.width-4))
	}

	return line
}

// renderOverlay renders the approval action overlay
func (av *ApprovalView) renderOverlay() string {
	// Title
	title := approvalOverlayTitleStyle.Render(fmt.Sprintf("Approve %s?", av.artifactName))

	// Description
	desc := approvalOverlayDescStyle.Render("Review the artifact above. Once approved, you'll proceed to the next phase.")

	// Action buttons hint
	actions := lipgloss.JoinHorizontal(
		lipgloss.Center,
		approvalKeyStyle.Render("Y")+approvalActionStyle.Render(" Approve & Continue"),
		approvalSepStyle.Render("  │  "),
		approvalKeyStyle.Render("E")+approvalActionStyle.Render(" Edit"),
		approvalSepStyle.Render("  │  "),
		approvalKeyStyle.Render("B")+approvalActionStyle.Render(" Go Back"),
		approvalSepStyle.Render("  │  "),
		approvalKeyStyle.Render("↑/↓")+approvalActionStyle.Render(" Scroll"),
	)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		desc,
		"",
		actions,
	)

	return approvalOverlayStyle.
		Width(av.width).
		Render(content)
}

// Styles for approval view
var (
	approvalContentStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1)

	approvalContentHeaderStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Padding(0, 0, 1, 0)

	approvalH1Style = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	approvalH2Style = lipgloss.NewStyle().
		Foreground(theme.Info).
		Bold(true)

	approvalH3Style = lipgloss.NewStyle().
		Foreground(theme.Text).
		Bold(true)

	approvalBulletStyle = lipgloss.NewStyle().
		Foreground(theme.Primary)

	approvalCheckboxStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	approvalCheckboxDoneStyle = lipgloss.NewStyle().
		Foreground(theme.Success)

	approvalStrikethroughStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	approvalCodeBlockStyle = lipgloss.NewStyle().
		Foreground(theme.Warning)

	approvalDividerStyle = lipgloss.NewStyle().
		Foreground(theme.Border)

	approvalOverlayStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Primary).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		Padding(1, 2).
		Align(lipgloss.Center)

	approvalOverlayTitleStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Align(lipgloss.Center)

	approvalOverlayDescStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Align(lipgloss.Center)

	approvalKeyStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	approvalActionStyle = lipgloss.NewStyle().
		Foreground(theme.Text)

	approvalSepStyle = lipgloss.NewStyle().
		Foreground(theme.Border)
)
