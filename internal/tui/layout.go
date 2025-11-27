package tui

import tea "github.com/charmbracelet/bubbletea"

// LayoutCache stores cached component heights to avoid redundant rendering
type LayoutCache struct {
	HeaderHeight int
	FooterHeight int
	LastWidth    int
	LastHeight   int
	Dirty        bool
}

// ContentProvider provides content without header/footer for embedding in workflow
type ContentProvider interface {
	tea.Model
	ViewContent() string
}