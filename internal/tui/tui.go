package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// Run starts the TUI with the given session and event bus
func Run(session *types.Session, bus *events.EventBus) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	if bus == nil {
		return fmt.Errorf("event bus cannot be nil")
	}

	// Create model
	model := NewModel(session, bus)

	// Create Bubbletea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if session was paused
	if m, ok := finalModel.(Model); ok && m.paused {
		return &PausedError{}
	}

	return nil
}

// PausedError indicates the TUI was exited with Ctrl+C (pause signal)
type PausedError struct{}

func (e *PausedError) Error() string {
	return "session paused by user"
}

// IsPausedError checks if an error is a PausedError
func IsPausedError(err error) bool {
	_, ok := err.(*PausedError)
	return ok
}

// RunSpecifyMode starts the TUI in specify mode
func RunSpecifyMode(session *types.SpecifySession) error {
	if session == nil {
		return fmt.Errorf("specify session cannot be nil")
	}

	// Create specify model
	model := NewSpecifyModel(session)

	// Create Bubbletea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("specify TUI error: %w", err)
	}

	return nil
}

// RunPlanMode starts the TUI in plan mode
func RunPlanMode(session *types.PlanSession) error {
	if session == nil {
		return fmt.Errorf("plan session cannot be nil")
	}

	// Create plan model
	model := NewPlanModel(session)

	// Create Bubbletea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("plan TUI error: %w", err)
	}

	return nil
}

// RunWorkflowMode starts the TUI in unified workflow mode
func RunWorkflowMode(session *types.WorkflowSession) error {
	if session == nil {
		return fmt.Errorf("workflow session cannot be nil")
	}

	// Create workflow model
	model := NewWorkflowModel(session)

	// Create Bubbletea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("workflow TUI error: %w", err)
	}

	return nil
}
