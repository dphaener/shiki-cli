package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)

// DisplayPhase represents a phase in the progress stepper with flexible naming and status
type DisplayPhase struct {
	Name       string
	Status     types.WorkflowPhaseStatus
}

// ProgressStepper displays workflow progress through phases
type ProgressStepper struct {
	phases []DisplayPhase
	width  int
}

// NewProgressStepper creates a new progress stepper
func NewProgressStepper(phases []DisplayPhase) ProgressStepper {
	return ProgressStepper{
		phases: phases,
		width:  80, // default
	}
}

// SetWidth sets the width of the stepper
func (p *ProgressStepper) SetWidth(width int) {
	p.width = width
}

// SetPhases updates the phases and their statuses
func (p *ProgressStepper) SetPhases(phases []DisplayPhase) {
	p.phases = phases
}

// View renders the progress stepper
func (p ProgressStepper) View() string {
	if len(p.phases) == 0 {
		return ""
	}

	var parts []string
	connector := stepperConnectorStyle.Render(" → ")

	for i, phase := range p.phases {
		// Render step with status indicator
		step := p.renderStep(i+1, phase.Name, phase.Status)
		parts = append(parts, step)

		// Add connector between steps (not after last)
		if i < len(p.phases)-1 {
			parts = append(parts, connector)
		}
	}

	// Join all parts
	stepperContent := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	// Center in available width
	return stepperContainerStyle.
		Width(p.width).
		Render(stepperContent)
}

// renderStep renders a single step with number and name
func (p ProgressStepper) renderStep(number int, name string, status types.WorkflowPhaseStatus) string {
	var style lipgloss.Style
	var indicator string

	switch status {
	case types.PhaseStatusComplete:
		style = stepCompleteStyle
		indicator = "✓"
	case types.PhaseStatusCurrent:
		style = stepCurrentStyle
		indicator = "●"
	default: // pending
		style = stepPendingStyle
		indicator = "○"
	}

	// Format: [indicator] number. Name
	stepText := indicator + " " + name
	return style.Render(stepText)
}

// Styles for progress stepper
var (
	stepperContainerStyle = lipgloss.NewStyle().
		Padding(0, 1).
		AlignHorizontal(lipgloss.Center)

	stepperConnectorStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	stepCompleteStyle = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(false)

	stepCurrentStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	stepPendingStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Bold(false)
)
