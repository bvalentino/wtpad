package tui

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

type activeTab int

const (
	tabTodos activeTab = iota
	tabNotes
)

type appMode int

const (
	modeNormal appMode = iota
	modeInput
	modeEditor
	modeHelp
)

// Layout constants
const (
	tabStripHeight = 3
	footerHeight   = 1
	sideBorderSize = 2 // left + right │ borders
)

// ASCII art header, 6 lines.
const asciiHeader = `  ___       _________              _________
  __ |     / /__  __/_____________ ______  /
  __ | /| / /__  /  ___  __ \  __ ` + "`" + `/  __  /
  __ |/ |/ / _  /   __  /_/ / /_/ // /_/ /
  ____/|__/  /_/    _  .___/\__,_/ \__,_/
                    /_/`

const asciiHeaderHeight = 6

type App struct {
	store     *store.Store
	width     int
	height    int
	activeTab activeTab
	mode      appMode

	// Pre-computed layout dimensions (set in layoutVertical)
	showFullHeader bool
	headerHeight   int
	contentHeight  int
	contentWidth   int

	// Pane metadata
	branch string

	todosPane  todosModel
	notesPane  notesModel
	editorPane editorModel
	helpPane   helpModel
}

func New(s *store.Store, todos []model.Todo, notes []model.Note, branch string) App {
	tp := newTodos(todos, s)
	np := newNotes(notes, s)
	return App{
		store:     s,
		activeTab: tabTodos,
		mode:      modeNormal,
		branch:    branch,
		todosPane: tp,
		notesPane: np,
		editorPane: newEditorModel(s),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case enterInputMsg:
		a.mode = modeInput
		return a, nil
	case exitInputMsg:
		a.mode = modeNormal
		return a, nil
	case enterEditorMsg:
		m := msg.(enterEditorMsg)
		a.editorPane = a.editorPane.openEditor(m.name, m.body, a.contentWidth, a.contentHeight)
		a.mode = modeEditor
		return a, nil
	case saveNoteMsg:
		notes, err := a.store.ListNotes()
		if err != nil {
			log.Printf("wtpad: failed to list notes after save: %v", err)
		} else {
			a.notesPane = a.notesPane.SetNotes(notes)
		}
		a.mode = modeNormal
		return a, nil
	case exitEditorMsg:
		a.mode = modeNormal
		return a, nil
	case exitHelpMsg:
		a.mode = modeNormal
		return a, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a = a.layoutVertical()
		if a.mode == modeEditor {
			a.editorPane = a.editorPane.resize(a.contentWidth, a.contentHeight)
		}
		if a.mode == modeHelp {
			a.helpPane.width = msg.Width
			a.helpPane.height = msg.Height
		}
		return a, nil

	case tea.KeyMsg:
		// ctrl+c always quits, regardless of mode
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		// In normal mode, handle global keys
		if a.mode == modeNormal {
			switch msg.String() {
			case "t":
				if a.activeTab != tabTodos {
					a = a.switchTab(tabTodos)
					return a, nil
				}
				// On todos tab, 't' falls through to pane
			case "n":
				if a.activeTab != tabNotes {
					a = a.switchTab(tabNotes)
					return a, nil
				}
				// On notes tab, 'n' falls through to pane (new note)
			case "?":
				a.helpPane.width = a.width
				a.helpPane.height = a.height
				a.mode = modeHelp
				return a, nil
			case "q":
				return a, tea.Quit
			}
		}
	}

	// Delegate to help overlay when in help mode
	if a.mode == modeHelp {
		var cmd tea.Cmd
		a.helpPane, cmd = a.helpPane.Update(msg)
		return a, cmd
	}

	// Delegate to editor when in editor mode
	if a.mode == modeEditor {
		var cmd tea.Cmd
		a.editorPane, cmd = a.editorPane.Update(msg)
		return a, cmd
	}

	// Delegate to active tab's pane
	var cmd tea.Cmd
	switch a.activeTab {
	case tabTodos:
		a.todosPane, cmd = a.todosPane.Update(msg)
	case tabNotes:
		a.notesPane, cmd = a.notesPane.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	if a.mode == modeHelp {
		return a.helpPane.View()
	}
	// Before the first WindowSizeMsg, dimensions are zero.
	if a.width == 0 || a.height == 0 {
		return ""
	}

	var sections []string

	// Header
	sections = append(sections, a.renderHeader())

	// Tab strip
	sections = append(sections, a.renderTabStrip())

	// Content area with side borders
	sections = append(sections, a.renderContent())

	// Footer
	sections = append(sections, a.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader returns the ASCII art or compact header.
func (a App) renderHeader() string {
	if a.showFullHeader {
		return headerStyle.Render(asciiHeader)
	}
	compact := "wtpad"
	if a.branch != "" {
		compact += " · " + a.branch
	}
	return headerStyle.Render(compact)
}

// renderTabStrip returns the 3-line tab chrome.
func (a App) renderTabStrip() string {
	todoLabel := " Todo (t) "
	noteLabel := " Notes (n) "
	w := a.width

	if a.activeTab == tabTodos {
		return a.renderTabStripLeft(todoLabel, noteLabel, w)
	}
	return a.renderTabStripRight(todoLabel, noteLabel, w)
}

// renderTabStripLeft renders tab strip with the left tab (Todos) active.
func (a App) renderTabStripLeft(activeLabel, inactiveLabel string, w int) string {
	activeLabelWidth := lipgloss.Width(activeLabel)
	inactiveLabelDisplay := tabInactive.Render(inactiveLabel)

	// Line 1: ┌──────────┐
	line1Top := "┌" + strings.Repeat("─", activeLabelWidth) + "┐"

	// Line 2: │ Todo (t) │ Notes (n)
	line2 := "│" + tabActive.Render(activeLabel) + "│ " + inactiveLabelDisplay

	// Line 3: │          └───────────┐
	// Width: "│" (1) + spaces (activeLabelWidth) + "└" (1) + dashes + "┐" (1) = w
	remaining := w - activeLabelWidth - 3
	if remaining < 0 {
		remaining = 0
	}
	line3 := "│" + strings.Repeat(" ", activeLabelWidth) + "└" + strings.Repeat("─", remaining) + "┐"

	return line1Top + "\n" + line2 + "\n" + line3
}

// renderTabStripRight renders tab strip with the right tab (Notes) active.
func (a App) renderTabStripRight(inactiveLabel, activeLabel string, w int) string {
	activeLabelWidth := lipgloss.Width(activeLabel)
	inactiveLabelDisplay := tabInactive.Render(inactiveLabel)
	inactiveLabelWidth := lipgloss.Width(inactiveLabelDisplay)

	// Position: inactive label on left, then active tab box on right
	// The active tab box starts after the inactive label + spacing
	leftPad := inactiveLabelWidth + 2 // leading " " + label + " " before │

	// Line 1: spaces + ┌──────────┐
	line1 := strings.Repeat(" ", leftPad) + "┌" + strings.Repeat("─", activeLabelWidth) + "┐"

	// Line 2: " Todo (t) │ Notes (n)│"
	line2 := " " + inactiveLabelDisplay + " │" + tabActive.Render(activeLabel) + "│"

	// Line 3: ┌────────────┘           └─────────────────────┐
	// ┘ aligns with left │ of tab box, └ aligns with right │
	// Content top border wraps around the tab opening.
	// "┌" (1) + dashes (leftPad-1) + "┘" (1) + spaces (activeLabelWidth) + "└" (1) + dashes + "┐" (1) = w
	leftFill := leftPad - 1
	if leftFill < 0 {
		leftFill = 0
	}
	rightFill := w - leftPad - activeLabelWidth - 3 // -3 for ┘, └, ┐
	if rightFill < 0 {
		rightFill = 0
	}
	line3 := "┌" + strings.Repeat("─", leftFill) + "┘" +
		strings.Repeat(" ", activeLabelWidth) +
		"└" + strings.Repeat("─", rightFill) + "┐"

	return line1 + "\n" + line2 + "\n" + line3
}

// renderContent renders the active tab's content with side borders.
func (a App) renderContent() string {
	var content string
	if a.mode == modeEditor {
		content = a.editorPane.View()
	} else {
		switch a.activeTab {
		case tabTodos:
			content = a.todosPane.View()
		case tabNotes:
			content = a.notesPane.View()
		}
	}

	// Split content into lines and pad/frame each with side borders
	lines := strings.Split(content, "\n")

	var b strings.Builder
	for i := 0; i < a.contentHeight; i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		// Pad line to content width
		lineWidth := lipgloss.Width(line)
		pad := a.contentWidth - lineWidth
		if pad < 0 {
			pad = 0
		}
		b.WriteString("│ " + line + strings.Repeat(" ", pad) + " │")
	}

	// Bottom border
	b.WriteString("\n")
	b.WriteString("└" + strings.Repeat("─", a.width-2) + "┘")

	return b.String()
}

// renderFooter returns the footer line with mode-aware hints.
func (a App) renderFooter() string {
	c := a.todosPane.Counts()
	parts := []string{fmt.Sprintf("%d open", c.Open)}
	if c.InProgress > 0 {
		parts = append(parts, fmt.Sprintf("%d in progress", c.InProgress))
	}
	parts = append(parts, fmt.Sprintf("%d done", c.Done))
	counts := strings.Join(parts, " · ")

	if msg := a.todosPane.StatusMsg(); msg != "" {
		return footerStyle.Render(counts + " · " + msg)
	}

	var hint string
	switch a.mode {
	case modeInput:
		hint = "enter confirm · esc cancel"
	case modeEditor:
		hint = a.editorPane.FooterHint()
	case modeHelp:
		hint = "esc close"
	default:
		hint = "? help · t/n switch"
	}

	return footerStyle.Render(counts + " · " + hint)
}

// layoutVertical computes dimensions for the vertical layout.
func (a App) layoutVertical() App {
	if a.height >= 30 {
		a.showFullHeader = true
		a.headerHeight = asciiHeaderHeight
	} else {
		a.showFullHeader = false
		a.headerHeight = 1
	}

	// contentHeight: total height minus header, tab strip, footer, bottom border
	a.contentHeight = a.height - a.headerHeight - tabStripHeight - footerHeight - 1 // -1 for bottom border └─┘
	if a.contentHeight < 1 {
		a.contentHeight = 1
	}

	a.contentWidth = a.width - sideBorderSize - 2 // -2 for the spaces after │
	if a.contentWidth < 1 {
		a.contentWidth = 1
	}

	a.todosPane = a.todosPane.SetSize(a.contentWidth, a.contentHeight)
	a.notesPane = a.notesPane.SetSize(a.contentWidth, a.contentHeight)
	return a
}

// switchTab switches to the given tab.
func (a App) switchTab(tab activeTab) App {
	a.activeTab = tab
	a.todosPane = a.todosPane.SetFocus(tab == tabTodos)
	a.notesPane = a.notesPane.SetFocus(tab == tabNotes)
	return a
}
