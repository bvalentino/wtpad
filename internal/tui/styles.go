package tui

import "github.com/charmbracelet/lipgloss"

var (
	selectionBg = lipgloss.Color("236")

	// Todo pane styles — one per status, selection bg composed at render time.
	todoSelected = lipgloss.NewStyle().
			Background(selectionBg)

	todoDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Strikethrough(true)

	todoInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	// Note pane styles
	noteSelected = lipgloss.NewStyle().
			Background(selectionBg)

	noteHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	notePreview = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	noteConfirm = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// Editor overlay styles
	editorHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	// Help overlay styles
	helpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	helpSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	helpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Tab strip styles
	tabActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	tabInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	// Header style
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62"))

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Hint style (dimmed inline hints like "Add (a)")
	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	// Divider style (between open/done todos)
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
