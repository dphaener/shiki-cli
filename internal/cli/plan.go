package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/tui"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewPlanCommand creates the plan command
func NewPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan [spec-slug]",
		Short: "Create implementation plan from completed specification",
		Long: `Create an implementation plan through interactive AI collaboration.

The plan command requires a completed specification (status="complete").
If no spec slug is provided, you'll be prompted to select from available specs.

Examples:
  collab plan 001-user-authentication
  collab plan                           # Shows list of completed specs to select
  collab plan list                      # List all plans`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle no arguments - show spec picker
			if len(args) == 0 {
				return showSpecPicker()
			}

			specSlug := strings.Join(args, " ")

			// Check for list subcommand
			if specSlug == "list" {
				return listPlans()
			}

			// Start plan session with specified slug
			return startPlanSession(specSlug)
		},
	}

	return cmd
}

// startPlanSession creates and starts a new planning session
func startPlanSession(specSlug string) error {
	// Validate spec exists and is complete
	spec, err := storage.ValidateSpecForPlanning(specSlug)
	if err != nil {
		PrintError("%v", err)
		return ExitWithCode(ExitError)
	}

	// Setup plan session (creates plan.md if needed, contracts/ directory)
	result, err := storage.SetupPlanSession(specSlug)
	if err != nil {
		PrintError("Failed to setup plan session: %v", err)
		return ExitWithCode(ExitError)
	}

	// Check if resuming existing plan
	isResume := storage.PlanExists(specSlug)
	if isResume {
		PrintInfo("Resuming existing plan for: %s", spec.FeatureName)
	} else {
		PrintSuccess("Starting new plan for: %s", spec.FeatureName)
	}

	PrintSuccess("Feature number: %03d", spec.Number)
	PrintSuccess("Slug: %s", spec.Slug)
	PrintInfo("")
	PrintInfo("Launching interactive planning TUI...")

	// Load existing plan content if resuming
	var currentPlan string
	if isResume {
		currentPlan, _ = storage.LoadPlanContent(specSlug)
	}

	// Create session
	session := &types.PlanSession{
		ID:            fmt.Sprintf("plan-%d-%s", time.Now().Unix(), specSlug),
		SpecSlug:      specSlug,
		FeatureNumber: spec.Number,
		FriendlyName:  spec.FeatureName,
		Phase:         types.PlanPhaseInterrogation,
		ChatHistory:   []types.ChatMessage{},
		CurrentPlan:   currentPlan,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Status:        types.SessionRunning,
		SpecFile:      result.SpecFile,
		SpecDir:       result.SpecDir,
		PlanFile:      result.PlanFile,
		ContractsDir:  result.ContractsDir,
	}

	// Launch TUI in plan mode
	return tui.RunPlanMode(session)
}

// showSpecPicker displays an interactive list of completed specs for selection
func showSpecPicker() error {
	// Get completed specs
	specs, err := storage.GetCompletedSpecs()
	if err != nil {
		PrintError("Failed to list specs: %v", err)
		return ExitWithCode(ExitError)
	}

	if len(specs) == 0 {
		PrintInfo("No completed specifications found.")
		PrintInfo("Complete a specification first with: collab specify \"your feature\"")
		PrintInfo("Then mark it complete when done.")
		return nil
	}

	// Convert specs to list items
	items := make([]list.Item, len(specs))
	for i, spec := range specs {
		items[i] = specItem{spec: spec}
	}

	// Create list model
	l := list.New(items, specItemDelegate{}, 60, 14)
	l.Title = "Select a completed spec to plan"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true).
		Padding(0, 1)
	l.Styles.PaginationStyle = lipgloss.NewStyle().PaddingLeft(4)
	l.Styles.HelpStyle = lipgloss.NewStyle().PaddingLeft(4).PaddingBottom(1)

	m := specPickerModel{list: l}

	// Run the picker
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		PrintError("Picker error: %v", err)
		return ExitWithCode(ExitError)
	}

	// Check if user selected a spec
	if m, ok := finalModel.(specPickerModel); ok && m.selected != nil {
		return startPlanSession(m.selected.Slug)
	}

	return nil
}

// listPlans lists all existing plans
func listPlans() error {
	specs, err := storage.ListSpecs()
	if err != nil {
		PrintError("Failed to list specs: %v", err)
		return ExitWithCode(ExitError)
	}

	if len(specs) == 0 {
		PrintInfo("No specifications found.")
		PrintInfo("Create one with: collab specify \"your feature description\"")
		return nil
	}

	PrintInfo("Feature Plans:")
	PrintInfo("")

	hasPlans := false
	for _, spec := range specs {
		planPath := storage.GetPlanPath(spec.Slug)
		hasPlan := storage.PlanExists(spec.Slug)

		if hasPlan {
			hasPlans = true
			PrintInfo("  %03d - %s", spec.Number, spec.FeatureName)
			PrintInfo("      Slug: %s", spec.Slug)
			PrintInfo("      Plan: %s", planPath)
			PrintInfo("      Spec Status: %s", spec.Status)
			PrintInfo("")
		}
	}

	if !hasPlans {
		PrintInfo("No plans created yet.")
		PrintInfo("Create one with: collab plan <spec-slug>")
	} else {
		PrintInfo("To start planning: collab plan <spec-slug>")
	}

	return nil
}

// specItem represents a spec in the picker list
type specItem struct {
	spec *storage.FeatureSpec
}

func (i specItem) FilterValue() string { return i.spec.FeatureName }

// specItemDelegate handles rendering of spec items
type specItemDelegate struct{}

func (d specItemDelegate) Height() int                             { return 2 }
func (d specItemDelegate) Spacing() int                            { return 1 }
func (d specItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d specItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(specItem)
	if !ok {
		return
	}

	spec := i.spec
	selected := index == m.Index()

	// Styles
	normalStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("10")).
		Bold(true)
	dimStyle := lipgloss.NewStyle().
		PaddingLeft(4).
		Foreground(lipgloss.Color("240"))

	// Title line
	title := fmt.Sprintf("%03d - %s", spec.Number, spec.FeatureName)
	if selected {
		title = "> " + title
		fmt.Fprintln(w, selectedStyle.Render(title))
	} else {
		fmt.Fprintln(w, normalStyle.Render(title))
	}

	// Detail line
	hasPlan := storage.PlanExists(spec.Slug)
	planStatus := "No plan yet"
	if hasPlan {
		planStatus = "Has plan"
	}
	detail := fmt.Sprintf("%s | %s", spec.Slug, planStatus)
	fmt.Fprintln(w, dimStyle.Render(detail))
}

// specPickerModel is the model for the spec picker TUI
type specPickerModel struct {
	list     list.Model
	selected *storage.FeatureSpec
	quitting bool
}

func (m specPickerModel) Init() tea.Cmd {
	return nil
}

func (m specPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(specItem)
			if ok {
				m.selected = i.spec
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m specPickerModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// getSpecDir returns the spec directory path for display
func getSpecDir(slug string) string {
	return filepath.Join(storage.GetSpecsDir(), slug)
}
