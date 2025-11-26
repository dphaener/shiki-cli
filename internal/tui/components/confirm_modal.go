package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
)

// ConfirmModal displays a confirmation dialog overlay
type ConfirmModal struct {
	title       string
	message     string
	options     []string
	selected    int
	width       int
	height      int
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal(title, message string, options []string) *ConfirmModal {
	if len(options) == 0 {
		options = []string{"Cancel", "Confirm"}
	}

	return &ConfirmModal{
		title:    title,
		message:  message,
		options:  options,
		selected: 0,
		width:    50,
		height:   10,
	}
}

// SetSize sets the modal dimensions
func (m *ConfirmModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SelectNext moves selection to next option
func (m *ConfirmModal) SelectNext() {
	if m.selected < len(m.options)-1 {
		m.selected++
	}
}

// SelectPrev moves selection to previous option
func (m *ConfirmModal) SelectPrev() {
	if m.selected > 0 {
		m.selected--
	}
}

// GetSelected returns the index of the selected option
func (m *ConfirmModal) GetSelected() int {
	return m.selected
}

// GetSelectedOption returns the selected option text
func (m *ConfirmModal) GetSelectedOption() string {
	if m.selected >= 0 && m.selected < len(m.options) {
		return m.options[m.selected]
	}
	return ""
}

// View renders the modal
func (m ConfirmModal) View() string {
	// Title
	titleView := modalTitleStyle.Render(m.title)

	// Message (wrapped)
	messageView := modalMessageStyle.
		Width(m.width - 4).
		Render(m.message)

	// Options as buttons
	var optionParts []string
	for i, opt := range m.options {
		var style lipgloss.Style
		if i == m.selected {
			style = modalButtonSelectedStyle
		} else {
			style = modalButtonStyle
		}
		optionParts = append(optionParts, style.Render(opt))
	}
	optionsView := lipgloss.JoinHorizontal(lipgloss.Center, optionParts...)

	// Help text
	helpText := modalHelpStyle.Render("←/→ to select • Enter to confirm • Esc to cancel")

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleView,
		"",
		messageView,
		"",
		optionsView,
		"",
		helpText,
	)

	// Wrap in modal container
	return modalContainerStyle.
		Width(m.width).
		Render(content)
}

// GoBackConfirmModal creates a modal for confirming go-back action
func GoBackConfirmModal() *ConfirmModal {
	return NewConfirmModal(
		"Go Back?",
		"This will preserve current artifacts as drafts.\nYou'll return to the previous phase to make changes.",
		[]string{"Cancel", "Go Back"},
	)
}

// QuitConfirmModal creates a modal for confirming quit
func QuitConfirmModal() *ConfirmModal {
	return NewConfirmModal(
		"Quit Workflow?",
		"Your progress will be saved.\nYou can resume later with 'collab feature resume'.",
		[]string{"Cancel", "Save & Quit"},
	)
}

// Styles for modal
var (
	modalContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Background(theme.BgDark)

	modalTitleStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Align(lipgloss.Center)

	modalMessageStyle = lipgloss.NewStyle().
		Foreground(theme.Text).
		Align(lipgloss.Center)

	modalButtonStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 2).
		MarginRight(1)

	modalButtonSelectedStyle = lipgloss.NewStyle().
		Foreground(theme.Text).
		Background(theme.Primary).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 2).
		MarginRight(1).
		Bold(true)

	modalHelpStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Italic(true).
		Align(lipgloss.Center)
)
