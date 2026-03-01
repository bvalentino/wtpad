package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// exitHelpMsg signals root to leave help mode.
type exitHelpMsg struct{}

type helpModel struct {
	width  int
	height int
}

func (h helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return h, func() tea.Msg { return exitHelpMsg{} }
	}
	return h, nil
}

func (h helpModel) View() string {
	var b strings.Builder

	b.WriteString(helpTitle.Render("wtpad — keyboard shortcuts"))
	b.WriteString("\n\n")

	sections := []struct {
		name string
		keys []struct{ key, desc string }
	}{
		{"Global", []struct{ key, desc string }{
			{"t", "Todos tab"},
			{"n", "Notes tab"},
			{"?", "Toggle this help"},
			{"q / Ctrl+C", "Quit"},
		}},
		{"Todos", []struct{ key, desc string }{
			{"j / k", "Navigate"},
			{"a", "Add todo"},
			{"d / Space", "Toggle done"},
			{"p", "Toggle in progress"},
			{"Enter", "Edit selected"},
			{"x", "Delete selected"},
			{"D", "Delete all completed"},
			{"c", "Copy selected"},
		}},
		{"Notes", []struct{ key, desc string }{
			{"j / k", "Navigate"},
			{"n", "New note"},
			{"e / Enter", "Edit selected"},
			{"x", "Delete selected"},
		}},
		{"Editor", []struct{ key, desc string }{
			{"Ctrl+S", "Save"},
			{"Esc", "Discard / close"},
		}},
	}

	for i, s := range sections {
		b.WriteString(helpSection.Render(s.name))
		b.WriteString("\n")
		for _, k := range s.keys {
			key := helpKey.Width(12).Render(k.key)
			desc := helpDesc.Render(k.desc)
			b.WriteString("  " + key + desc + "\n")
		}
		if i < len(sections)-1 {
			b.WriteString("\n")
		}
	}

	content := b.String()

	// Center the content block in the terminal.
	contentWidth := lipgloss.Width(content)
	contentHeight := strings.Count(content, "\n") + 1

	padLeft := 0
	if h.width > contentWidth {
		padLeft = (h.width - contentWidth) / 2
	}
	padTop := 0
	if h.height > contentHeight {
		padTop = (h.height - contentHeight) / 2
	}

	return lipgloss.NewStyle().
		PaddingLeft(padLeft).
		PaddingTop(padTop).
		Width(h.width).
		Height(h.height).
		Render(content)
}
