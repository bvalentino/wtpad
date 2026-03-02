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
	modeTemplate
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

	todosPane    todosModel
	notesPane    notesModel
	editorPane   editorModel
	helpPane     helpModel
	templatePane templateModal
}

func New(s *store.Store, ts *store.TemplateStore, todos []model.Todo, notes []model.Note, branch string) App {
	tp := newTodos(todos, s)
	np := newNotes(notes, s)
	return App{
		store:        s,
		activeTab:    tabTodos,
		mode:         modeNormal,
		branch:       branch,
		todosPane:    tp,
		notesPane:    np,
		editorPane:   newEditorModel(s),
		templatePane: newTemplateModal(ts),
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
	case enterTemplateMsg:
		m := msg.(enterTemplateMsg)
		a.templatePane = a.templatePane.open(m.saving, a.width, a.height)
		a.mode = modeTemplate
		return a, nil
	case importTemplateMsg:
		m := msg.(importTemplateMsg)
		a.todosPane = a.todosPane.ImportTodos(m.todos)
		a.mode = modeNormal
		return a, nil
	case saveTemplateMsg:
		m := msg.(saveTemplateMsg)
		openTodos := a.todosPane.OpenTodos()
		if _, err := a.templatePane.tstore.SaveTemplate(m.name, openTodos); err != nil {
			log.Printf("wtpad: failed to save template: %v", err)
		}
		a.mode = modeNormal
		return a, nil
	case exitTemplateMsg:
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
		if a.mode == modeTemplate {
			a.templatePane = a.templatePane.resize(msg.Width, msg.Height)
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

	// Delegate to template modal when in template mode
	if a.mode == modeTemplate {
		var cmd tea.Cmd
		a.templatePane, cmd = a.templatePane.Update(msg)
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
	if a.mode == modeTemplate {
		return a.templatePane.View()
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
// Lines 1-2 (top border + label) are rendered by lipgloss borders.
// Line 3 (connection to content area) is manually constructed so that
// the tab opening flows seamlessly into the content box.
func (a App) renderTabStrip() string {
	todoLabel := "Todo (t)"
	noteLabel := "Notes (n)"

	var todoTab, noteTab string
	if a.activeTab == tabTodos {
		todoTab = activeTabStyle.Render(todoLabel)
		noteTab = inactiveTabStyle.Render(noteLabel)
	} else {
		todoTab = inactiveTabStyle.Render(todoLabel)
		noteTab = activeTabStyle.Render(noteLabel)
	}

	// Lines 1-2: join the two tabs (each 2 lines: top border + label row)
	row := lipgloss.JoinHorizontal(lipgloss.Top, todoTab, noteTab)

	// Pad lines 1-2 to terminal width
	rowLines := strings.Split(row, "\n")
	for i, l := range rowLines {
		if pad := a.width - lipgloss.Width(l); pad > 0 {
			rowLines[i] = l + strings.Repeat(" ", pad)
		}
	}

	// Line 3: content top border with opening under the active tab.
	// Each tab's display width includes left border (1) + padding (1) +
	// label + padding (1) + right border (1). Inner width = displayW - 2.
	todoW := lipgloss.Width(todoTab)
	noteW := lipgloss.Width(noteTab)
	todoInner := todoW - 2
	noteInner := noteW - 2

	gapFill := a.width - todoW - noteW - 1 // -1 for the ╮ at right edge
	if gapFill < 0 {
		gapFill = 0
	}

	var line3 string
	if a.activeTab == tabTodos {
		// │<spaces>╰┴<dashes>┴<dashes>╮
		line3 = "│" + strings.Repeat(" ", todoInner) +
			"╰┴" + strings.Repeat("─", noteInner) + "┴" +
			strings.Repeat("─", gapFill) + "╮"
	} else {
		// ├<dashes>┴╯<spaces>╰<dashes>╮
		line3 = "├" + strings.Repeat("─", todoInner) +
			"┴╯" + strings.Repeat(" ", noteInner) + "╰" +
			strings.Repeat("─", gapFill) + "╮"
	}
	line3 = dimBorder.Render(line3)

	return strings.Join(rowLines, "\n") + "\n" + line3
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
		b.WriteString(dimBorder.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + dimBorder.Render("│"))
	}

	// Bottom border
	b.WriteString("\n")
	b.WriteString(dimBorder.Render("╰" + strings.Repeat("─", a.width-2) + "╯"))

	return b.String()
}

// renderFooter returns the footer line with mode-aware hints.
func (a App) renderFooter() string {
	c := a.todosPane.Counts()
	var counts string
	if a.todosPane.ShowingCompleted() {
		counts = fmt.Sprintf("viewing %d done", c.Done)
	} else {
		parts := []string{fmt.Sprintf("%d open", c.Open)}
		if c.InProgress > 0 {
			parts = append(parts, fmt.Sprintf("%d in progress", c.InProgress))
		}
		parts = append(parts, fmt.Sprintf("%d done", c.Done))
		counts = strings.Join(parts, " · ")
	}

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
	case modeTemplate:
		hint = a.templatePane.FooterHint()
	default:
		hint = a.todosPane.FooterHint()
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
