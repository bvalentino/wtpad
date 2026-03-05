package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// enterHelpMsg signals root to enter help mode from an overlay.
type enterHelpMsg struct{}

// exitHelpMsg signals root to leave help mode.
type exitHelpMsg struct{}

type helpModel struct {
	lines        []string
	scrollOffset int
	width        int
	height       int
}

func (m helpModel) open(w, h int) helpModel {
	m.scrollOffset = 0
	return m.resize(w, h)
}

func (m helpModel) resize(w, h int) helpModel {
	m.width = w
	m.height = h
	m.lines = m.buildLines()
	m = m.clampScroll()
	return m
}

func (h helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return h, nil
	}

	contentH := overlayContentHeight(h.height)

	switch keyMsg.String() {
	case "esc", "q", "?":
		return h, func() tea.Msg { return exitHelpMsg{} }
	case "up", "k":
		if h.scrollOffset > 0 {
			h.scrollOffset--
		}
	case "down", "j":
		h.scrollOffset++
		h = h.clampScroll()
	case "pgup":
		h.scrollOffset -= contentH
		if h.scrollOffset < 0 {
			h.scrollOffset = 0
		}
	case "pgdown":
		h.scrollOffset += contentH
		h = h.clampScroll()
	}

	return h, nil
}

func (h helpModel) View() string {
	if h.width == 0 || h.height == 0 {
		return ""
	}

	visible := h.lines
	if h.scrollOffset < len(visible) {
		visible = visible[h.scrollOffset:]
	} else {
		visible = nil
	}

	return renderOverlayBox("Keyboard Shortcuts", visible, h.width, h.height, h.FooterHint())
}

// FooterHint returns the contextual hint for the help overlay.
func (h helpModel) FooterHint() string {
	return "Scroll (↑↓) · Close (esc)"
}

func (h helpModel) clampScroll() helpModel {
	maxOffset := len(h.lines) - overlayContentHeight(h.height)
	if maxOffset < 0 {
		maxOffset = 0
	}
	if h.scrollOffset > maxOffset {
		h.scrollOffset = maxOffset
	}
	return h
}

func (h helpModel) buildLines() []string {
	sections := []struct {
		name string
		keys []struct{ key, desc string }
	}{
		{"Global", []struct{ key, desc string }{
			{"Tab", "Switch tab"},
			{"t", "Set title"},
			{"?", "Toggle this help"},
			{"q / Ctrl+C", "Quit"},
		}},
		{"Todos", []struct{ key, desc string }{
			{"↑ / ↓", "Navigate"},
			{"a", "Add todo"},
			{"Space", "Toggle done"},
			{"i", "Toggle in progress"},
			{"v", "View completed"},
			{"Enter", "Edit selected"},
			{"J / K", "Move todo down / up"},
			{"x", "Delete selected"},
			{"X", "Clear all in view"},
			{"c", "Copy selected"},
			{"T", "Import template"},
			{"S", "Save as template"},
		}},
		{"Notes", []struct{ key, desc string }{
			{"↑ / ↓", "Navigate"},
			{"a", "New note"},
			{"Enter", "View selected"},
			{"e", "Edit selected"},
			{"x", "Delete selected"},
		}},
		{"Prompts", []struct{ key, desc string }{
			{"↑ / ↓", "Navigate"},
			{"c", "Copy to clipboard"},
			{"Enter", "View selected"},
			{"a", "New prompt"},
			{"e", "Edit selected"},
			{"x", "Delete selected"},
		}},
		{"AI", []struct{ key, desc string }{
			{"↑ / ↓", "Navigate"},
			{"c", "Copy selected"},
			{"X", "Clear all"},
		}},
		{"Viewer / Editor", []struct{ key, desc string }{
			{"e", "Edit (viewer)"},
			{"Ctrl+S", "Save (editor)"},
			{"Esc", "Back / discard"},
		}},
	}

	var lines []string

	for i, s := range sections {
		lines = append(lines, helpSection.Render(s.name))
		for _, k := range s.keys {
			key := helpKey.Width(12).Render(k.key)
			desc := helpDesc.Render(k.desc)
			lines = append(lines, "  "+key+desc)
		}
		if i < len(sections)-1 {
			lines = append(lines, "")
		}
	}

	return lines
}
