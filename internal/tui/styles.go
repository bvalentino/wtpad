package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Overlay box layout constants.
const (
	overlayBorderWidth = 2 // left + right │
	overlayPadWidth    = 2 // 1 space after each │
	overlayBoxChrome   = overlayBorderWidth + overlayPadWidth
	overlayBoxLines    = 2 // top border + bottom border
)

// overlayContentWidth returns the usable width inside the overlay box.
func overlayContentWidth(termWidth int) int {
	w := termWidth - overlayBoxChrome
	if w < 1 {
		w = 1
	}
	return w
}

// overlayTopBorder builds ╭─── Title ───╮ with the title centered.
func overlayTopBorder(title string, width int) string {
	styled := overlayTitle.Render(title)
	titleW := lipgloss.Width(styled)

	// Available space for dashes: width minus ╭, ╮, and the styled title with surrounding spaces.
	inner := width - 2 // excluding ╭ and ╮
	if titleW == 0 {
		fill := inner
		if fill < 0 {
			fill = 0
		}
		return dimBorder.Render("╭" + strings.Repeat("─", fill) + "╮")
	}

	// " Title " with a space on each side of the styled text
	decorated := " " + styled + " "
	decoratedW := titleW + 2

	dashSpace := inner - decoratedW
	if dashSpace < 0 {
		dashSpace = 0
	}
	left := dashSpace / 2
	right := dashSpace - left

	return dimBorder.Render("╭"+strings.Repeat("─", left)) +
		decorated +
		dimBorder.Render(strings.Repeat("─", right)+"╮")
}

// overlayBottomBorder builds ╰──────╯ at the given width.
func overlayBottomBorder(width int) string {
	fill := width - 2
	if fill < 0 {
		fill = 0
	}
	return dimBorder.Render("╰" + strings.Repeat("─", fill) + "╯")
}

// overlayContentLine wraps a single line with │ borders and padding.
// Lines exceeding contentWidth are truncated as a safety backstop.
func overlayContentLine(line string, contentWidth int) string {
	lineW := lipgloss.Width(line)
	if lineW > contentWidth {
		line = truncate(line, contentWidth)
		lineW = lipgloss.Width(line)
	}
	pad := contentWidth - lineW
	return dimBorder.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + dimBorder.Render("│")
}

// overlayContentHeight returns the number of content lines inside an overlay box.
func overlayContentHeight(termHeight int) int {
	h := termHeight - overlayBoxLines - 1 // top border + bottom border + footer
	if h < 1 {
		h = 1
	}
	return h
}

// renderOverlayBox assembles a bordered overlay: top border, content lines, bottom border, footer.
func renderOverlayBox(title string, bodyLines []string, width, height int, footer string) string {
	cw := overlayContentWidth(width)
	contentH := overlayContentHeight(height)

	parts := make([]string, 0, contentH+3)
	parts = append(parts, overlayTopBorder(title, width))

	for i := 0; i < contentH; i++ {
		line := ""
		if i < len(bodyLines) {
			line = bodyLines[i]
		}
		parts = append(parts, overlayContentLine(line, cw))
	}

	parts = append(parts, overlayBottomBorder(width))
	parts = append(parts, footerStyle.Render(footer))

	return strings.Join(parts, "\n")
}

var (
	// Todo pane styles — one per status, selection composed at render time.
	todoSelected = lipgloss.NewStyle().
			Reverse(true)

	todoDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Strikethrough(true)

	todoInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	// List pane styles (shared by notes and prompts)
	listSelected = lipgloss.NewStyle().
			Reverse(true)

	listHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	listPreview = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	listConfirm = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// Overlay title style (rendered inside the top border)
	overlayTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	// Help overlay styles (same as overlay title)
	helpTitle = overlayTitle

	helpSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	helpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Border color styles for manually constructed chrome
	dimBorder = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	// Tab strip border (top + sides only; bottom connection line is manual)
	tabBorder = lipgloss.Border{
		Top:     "─",
		Left:    "│",
		Right:   "│",
		TopLeft: "╭", TopRight: "╮",
	}

	// Tab strip styles (no bottom border — connection line is built in renderTabStrip)
	activeTabStyle = lipgloss.NewStyle().
			Border(tabBorder, true, true, false, true).
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Border(tabBorder, true, true, false, true).
				BorderForeground(lipgloss.Color("240")).
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)

	// Title style (bold white, rendered above the logo)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

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

	// Template modal styles
	templateHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	templateSelected = lipgloss.NewStyle().
				Reverse(true)
)

// renderEmptyState renders centered lines (vertically and horizontally)
// within the given width and height.
func renderEmptyState(width, height int, lines []string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
		strings.Join(lines, "\n"))
}
