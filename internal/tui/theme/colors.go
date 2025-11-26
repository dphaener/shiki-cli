// Package theme provides the Sekkei Design System color palette and semantic color aliases
// for the Collab TUI application.
package theme

import "github.com/charmbracelet/lipgloss"

// Sekkei Design System - Emerald Palette
var (
	// Primary - Emerald Green
	Emerald50  = lipgloss.Color("#ecfdf5")
	Emerald100 = lipgloss.Color("#d1fae5")
	Emerald200 = lipgloss.Color("#a7f3d0")
	Emerald300 = lipgloss.Color("#6ee7b7")
	Emerald400 = lipgloss.Color("#34d399")
	Emerald500 = lipgloss.Color("#10b981")
	Emerald600 = lipgloss.Color("#059669")
	Emerald700 = lipgloss.Color("#047857")
	Emerald800 = lipgloss.Color("#065f46")
	Emerald900 = lipgloss.Color("#064e3b")

	// Neutrals - Slate
	Slate50  = lipgloss.Color("#f8fafc")
	Slate100 = lipgloss.Color("#f1f5f9")
	Slate200 = lipgloss.Color("#e2e8f0")
	Slate300 = lipgloss.Color("#cbd5e1")
	Slate400 = lipgloss.Color("#94a3b8")
	Slate500 = lipgloss.Color("#64748b")
	Slate600 = lipgloss.Color("#475569")
	Slate700 = lipgloss.Color("#334155")
	Slate800 = lipgloss.Color("#1e293b")
	Slate900 = lipgloss.Color("#0f172a")

	// Accent - Sky Blue
	Sky400 = lipgloss.Color("#38bdf8")
	Sky500 = lipgloss.Color("#0ea5e9")
	Sky600 = lipgloss.Color("#0284c7")

	// Amber for warnings
	Amber500 = lipgloss.Color("#f59e0b")

	// Red for errors
	Red500 = lipgloss.Color("#ef4444")
)

// Semantic color aliases for common use cases
var (
	// Brand colors
	Primary     = Emerald500
	PrimaryDark = Emerald600

	// Status colors
	Success      = Emerald600
	SuccessLight = Emerald100
	Info         = Sky500
	InfoLight    = lipgloss.Color("#e0f2fe")
	Warning      = Amber500
	WarningLight = lipgloss.Color("#fef3c7")
	Error        = Red500
	ErrorLight   = lipgloss.Color("#fee2e2")

	// Text colors
	Text      = Slate50
	TextMuted = Slate400
	TextDim   = Slate500

	// UI element colors
	Border       = Slate600
	BorderActive = Emerald500
	BgDark       = Slate900
	BgMedium     = Slate800
	BgLight      = Slate700
)
