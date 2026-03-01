package tui

import "github.com/charmbracelet/lipgloss"

var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	unfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Todo pane styles
	todoSelected = lipgloss.NewStyle().
			Background(lipgloss.Color("236"))

	todoDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Strikethrough(true)

	todoDoneSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Strikethrough(true).
				Background(lipgloss.Color("236"))

	// Note pane styles
	noteSelected = lipgloss.NewStyle().
			Background(lipgloss.Color("236"))

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

	editorFooter = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	editorConfirm = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

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

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
)
