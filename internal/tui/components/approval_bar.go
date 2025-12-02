package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// ApprovalBar displays context-sensitive keyboard shortcuts in the footer
type ApprovalBar struct {
	phase        types.WorkflowPhase
	width        int
	waitingForAI bool
}

// NewApprovalBar creates a new approval bar
func NewApprovalBar(phase types.WorkflowPhase) ApprovalBar {
	return ApprovalBar{
		phase: phase,
		width: 80, // default
	}
}

// SetWidth sets the width of the bar
func (a *ApprovalBar) SetWidth(width int) {
	a.width = width
}

// SetPhase updates the current phase
func (a *ApprovalBar) SetPhase(phase types.WorkflowPhase) {
	a.phase = phase
}

// SetWaitingForAI sets the AI waiting state
func (a *ApprovalBar) SetWaitingForAI(waiting bool) {
	a.waitingForAI = waiting
}

// View renders the approval bar
func (a ApprovalBar) View() string {
	shortcuts := a.getShortcuts()

	// Render each shortcut
	var parts []string
	for i, s := range shortcuts {
		parts = append(parts, a.renderShortcut(s.key, s.action))
		if i < len(shortcuts)-1 {
			parts = append(parts, approvalBarSeparator)
		}
	}

	shortcutStr := strings.Join(parts, "")

	// Loading state removed - now handled by ChatView component

	return approvalBarStyle.Width(a.width).Render(shortcutStr)
}

// shortcut represents a keyboard shortcut
type shortcut struct {
	key    string
	action string
}

// getShortcuts returns the shortcuts for the current phase
func (a ApprovalBar) getShortcuts() []shortcut {
	if types.IsApprovalPhase(a.phase) {
		shortcuts := []shortcut{
			{key: "Y", action: "Approve"},
			{key: "E", action: "Edit"},
		}
		if types.CanGoBack(a.phase) {
			shortcuts = append(shortcuts, shortcut{key: "B", action: "Go Back"})
		}
		shortcuts = append(shortcuts, shortcut{key: "Tab", action: "Switch Pane"})
		shortcuts = append(shortcuts, shortcut{key: "Ctrl+C", action: "Quit"})
		return shortcuts
	}

	// Working phase shortcuts
	return []shortcut{
		{key: "Enter", action: "Send"},
		{key: "Shift+Enter", action: "New Line"},
		{key: "Esc×2", action: "Clear"},
		{key: "Ctrl+D", action: "Done"},
		{key: "Tab", action: "Switch Pane"},
		{key: "↑/↓", action: "Scroll"},
		{key: "Esc", action: "Interrupt"},
		{key: "Ctrl+C", action: "Quit"},
	}
}

// renderShortcut renders a single shortcut
func (a ApprovalBar) renderShortcut(key, action string) string {
	return approvalBarKeyStyle.Render(key) + " " + approvalBarActionStyle.Render(action)
}

// Styles for approval bar
var (
	approvalBarStyle = lipgloss.NewStyle().
		Padding(0, 1)

	approvalBarKeyStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	approvalBarActionStyle = lipgloss.NewStyle().
		Foreground(theme.Text)

	approvalBarSeparator = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Render(" • ")

	approvalBarWaitingStyle = lipgloss.NewStyle().
		Foreground(theme.Warning)
)
