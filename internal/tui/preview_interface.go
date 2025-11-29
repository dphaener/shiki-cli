package tui

import tea "github.com/charmbracelet/bubbletea"

// PreviewComponent defines the interface for preview panes in phase models.
// All preview components (PlanPreview, TasksPreview, ImplementPreview, SpecPreview)
// implement this interface to enable generic phase model composition.
type PreviewComponent interface {
	// Init returns the initial command for the preview component.
	Init() tea.Cmd

	// View renders the preview component.
	View() string

	// SetSize updates the preview component's dimensions.
	SetSize(width, height int)

	// SetContent updates the preview content from a string (typically file content).
	SetContent(content string)
}

// PreviewUpdater wraps a PreviewComponent with its Update method.
// This is needed because Go interfaces can't express methods that return the implementing type.
// Each concrete preview type must provide an update function.
type PreviewUpdater interface {
	PreviewComponent

	// UpdatePreview processes a message and returns a command.
	// The preview component is modified in place (pointer receiver).
	UpdatePreview(msg tea.Msg) tea.Cmd
}

// PreviewWrapper wraps concrete preview types to implement PreviewUpdater.
// Usage: wrap := &PreviewWrapper[*components.PlanPreview]{Preview: &planPreview}
type PreviewWrapper[T PreviewComponent] struct {
	Preview   T
	UpdateFn  func(p T, msg tea.Msg) tea.Cmd
}

func (w *PreviewWrapper[T]) Init() tea.Cmd {
	return w.Preview.Init()
}

func (w *PreviewWrapper[T]) View() string {
	return w.Preview.View()
}

func (w *PreviewWrapper[T]) SetSize(width, height int) {
	w.Preview.SetSize(width, height)
}

func (w *PreviewWrapper[T]) SetContent(content string) {
	w.Preview.SetContent(content)
}

func (w *PreviewWrapper[T]) UpdatePreview(msg tea.Msg) tea.Cmd {
	if w.UpdateFn != nil {
		return w.UpdateFn(w.Preview, msg)
	}
	return nil
}
