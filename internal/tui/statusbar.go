package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type statusBarModel struct {
	dir       string
	branch    string
	openCount int
	doneCount int
	hint      string
}

func newStatusBar(dir, branch string) statusBarModel {
	return statusBarModel{
		dir:    dir,
		branch: branch,
		hint:   "? help · tab switch",
	}
}

func (m statusBarModel) View(width int) string {
	left := m.dir
	if m.branch != "" {
		left += " · " + m.branch
	}

	center := fmt.Sprintf("%d open · %d done", m.openCount, m.doneCount)

	right := m.hint

	// Calculate padding to fill the full width using display-width
	// (not byte-length) so multi-byte chars like · measure correctly.
	// Layout: " left   <pad>   center   <pad>   right "
	leftLen := lipgloss.Width(left)
	centerLen := lipgloss.Width(center)
	rightLen := lipgloss.Width(right)
	contentLen := leftLen + centerLen + rightLen + 4 // 4 for edge padding (2 each side)

	if width <= 0 {
		width = contentLen
	}

	gap := width - contentLen
	if gap < 2 {
		// Not enough room for padding; just join with single spaces
		line := " " + left + " " + center + " " + right + " "
		return statusBarStyle.Width(width).Render(line)
	}

	leftPad := gap / 2
	rightPad := gap - leftPad

	var b strings.Builder
	b.WriteString(" ")
	b.WriteString(left)
	b.WriteString(strings.Repeat(" ", leftPad+1))
	b.WriteString(center)
	b.WriteString(strings.Repeat(" ", rightPad+1))
	b.WriteString(right)
	b.WriteString(" ")

	return statusBarStyle.Width(width).Render(b.String())
}
