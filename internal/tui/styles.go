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
)
