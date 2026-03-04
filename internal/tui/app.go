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
	tabPrompts
)

type appMode int

const (
	modeNormal appMode = iota
	modeInput
	modeEditor
	modeViewer
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
	editorReturnMode appMode // mode to return to when exiting editor (viewer or normal)
	helpReturnMode   appMode // mode to return to when exiting help

	// Pre-computed layout dimensions (set in layoutVertical)
	showFullHeader bool
	headerHeight   int
	contentHeight  int
	contentWidth   int

	// Pane metadata
	branch string

	todosPane    todosModel
	notesPane    notesModel
	promptsPane  promptsModel
	editorPane   editorModel
	viewerPane   viewerModel
	helpPane     helpModel
	templatePane templateModal

	promptStore *store.PromptStore
}

func New(s *store.Store, ts *store.TemplateStore, ps *store.PromptStore, todos []model.Todo, notes []model.Note, prompts []model.Note, branch string) App {
	tp := newTodos(todos, s)
	np := newNotes(notes, s)
	pp := newPrompts(prompts, ps)
	return App{
		store:        s,
		promptStore:  ps,
		activeTab:    tabTodos,
		mode:         modeNormal,
		branch:       branch,
		todosPane:    tp,
		notesPane:    np,
		promptsPane:  pp,
		editorPane:   newEditorModel(),
		templatePane: newTemplateModal(ts),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case clearPromptStatusMsg:
		a.promptsPane, _ = a.promptsPane.Update(msg)
		return a, nil
	case clipboardResultMsg:
		var cmd tea.Cmd
		a.promptsPane, cmd = a.promptsPane.Update(msg)
		return a, cmd
	case enterInputMsg:
		a.mode = modeInput
		return a, nil
	case exitInputMsg:
		a.mode = modeNormal
		return a, nil
	case enterViewerMsg:
		m := msg.(enterViewerMsg)
		a.viewerPane = a.viewerPane.openViewer(m.name, m.body, a.width, a.height)
		a.mode = modeViewer
		return a, nil
	case exitViewerMsg:
		a.mode = modeNormal
		return a, nil
	case enterEditorMsg:
		m := msg.(enterEditorMsg)
		a.editorReturnMode = a.mode
		entityName := "Note"
		if a.activeTab == tabPrompts {
			entityName = "Prompt"
		}
		a.editorPane = a.editorPane.openEditor(m.name, m.body, entityName, a.activeTab, a.width, a.height)
		a.mode = modeEditor
		return a, nil
	case editorSaveMsg:
		a = a.handleEditorSave(msg.(editorSaveMsg))
		return a, nil
	case exitEditorMsg:
		if a.editorReturnMode == modeViewer {
			a.mode = modeViewer
		} else {
			a.mode = modeNormal
		}
		a.editorReturnMode = modeNormal
		return a, nil
	case enterHelpMsg:
		a.helpReturnMode = a.mode
		a.helpPane = a.helpPane.resize(a.width, a.height)
		a.mode = modeHelp
		return a, nil
	case exitHelpMsg:
		a.mode = a.helpReturnMode
		a.helpReturnMode = modeNormal
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
		// Resize all overlays so none hold stale dimensions when re-entered.
		a.editorPane = a.editorPane.resize(msg.Width, msg.Height)
		a.viewerPane = a.viewerPane.resize(msg.Width, msg.Height)
		a.helpPane = a.helpPane.resize(msg.Width, msg.Height)
		a.templatePane = a.templatePane.resize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// ctrl+c always quits, regardless of mode
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		// In normal mode, handle global keys
		if a.mode == modeNormal {
			switch msg.String() {
			case "tab":
				switch a.activeTab {
				case tabTodos:
					a = a.switchTab(tabNotes)
				case tabNotes:
					a = a.switchTab(tabPrompts)
				case tabPrompts:
					a = a.switchTab(tabTodos)
				}
				return a, nil
			case "?":
				return a, func() tea.Msg { return enterHelpMsg{} }
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

	// Delegate to viewer when in viewer mode
	if a.mode == modeViewer {
		var cmd tea.Cmd
		a.viewerPane, cmd = a.viewerPane.Update(msg)
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
	case tabPrompts:
		a.promptsPane, cmd = a.promptsPane.Update(msg)
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
	if a.mode == modeViewer {
		return a.viewerPane.View()
	}
	if a.mode == modeEditor {
		return a.editorPane.View()
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
// Lines 1-2 (top border + label) are rendered by lipgloss borders with a
// 1-space gap between tabs. Line 3 (connection to content area) is manually
// constructed so the active tab opening flows into the content box.
func (a App) renderTabStrip() string {
	todoLabel := "Todo"
	noteLabel := "Notes"
	promptLabel := "Prompts"

	renderTab := func(label string, active bool) string {
		if active {
			return activeTabStyle.Render(label)
		}
		return inactiveTabStyle.Render(label)
	}

	todoTab := renderTab(todoLabel, a.activeTab == tabTodos)
	noteTab := renderTab(noteLabel, a.activeTab == tabNotes)
	promptTab := renderTab(promptLabel, a.activeTab == tabPrompts)

	// Lines 1-2: join tabs with a 1-space gap
	gap := lipgloss.NewStyle().Width(1)
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		todoTab, gap.Render(" "), noteTab, gap.Render(" "), promptTab,
	)

	// Pad lines 1-2 to terminal width
	rowLines := strings.Split(row, "\n")
	for i, l := range rowLines {
		if pad := a.width - lipgloss.Width(l); pad > 0 {
			rowLines[i] = l + strings.Repeat(" ", pad)
		}
	}

	// Line 3: content top border with opening under the active tab.
	// Each tab's display width = left border(1) + pad(1) + label + pad(1) + right border(1).
	// Inner width = displayW - 2. The 1-space gaps become ─ on the connection line.
	todoW := lipgloss.Width(todoTab)
	noteW := lipgloss.Width(noteTab)
	promptW := lipgloss.Width(promptTab)
	todoInner := todoW - 2
	noteInner := noteW - 2
	promptInner := promptW - 2

	// tabsSpan = total columns consumed by tabs + gaps (2 gaps of 1 char each)
	tabsSpan := todoW + 1 + noteW + 1 + promptW

	fill := a.width - tabsSpan
	if fill < 0 {
		fill = 0
	}

	var line3 string
	switch a.activeTab {
	case tabTodos:
		line3 = "│" + strings.Repeat(" ", todoInner) +
			"╰─┴" + strings.Repeat("─", noteInner) +
			"┴─┴" + strings.Repeat("─", promptInner) +
			"┴" + strings.Repeat("─", fill) + "╮"
	case tabNotes:
		line3 = "├" + strings.Repeat("─", todoInner) +
			"┴─╯" + strings.Repeat(" ", noteInner) +
			"╰─┴" + strings.Repeat("─", promptInner) +
			"┴" + strings.Repeat("─", fill) + "╮"
	case tabPrompts:
		line3 = "├" + strings.Repeat("─", todoInner) +
			"┴─┴" + strings.Repeat("─", noteInner) +
			"┴─╯" + strings.Repeat(" ", promptInner) +
			"╰" + strings.Repeat("─", fill) + "╮"
	}
	line3 = dimBorder.Render(line3)

	return strings.Join(rowLines, "\n") + "\n" + line3
}

// renderContent renders the active tab's content with side borders.
func (a App) renderContent() string {
	var content string
	switch a.activeTab {
	case tabTodos:
		content = a.todosPane.View()
	case tabNotes:
		content = a.notesPane.View()
	case tabPrompts:
		content = a.promptsPane.View()
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
	var counts, hint string

	switch a.activeTab {
	case tabPrompts:
		counts = pluralize(a.promptsPane.count(), "prompt")
		if msg := a.promptsPane.StatusMsg(); msg != "" {
			return footerStyle.Render(counts + " · " + msg)
		}
		hint = "? help · tab switch · q quit"

	case tabNotes:
		counts = pluralize(a.notesPane.count(), "note")
		hint = "? help · tab switch · q quit"

	case tabTodos:
		c := a.todosPane.Counts()
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
		switch a.mode {
		case modeInput:
			hint = "enter confirm · esc cancel"
		case modeTemplate:
			hint = a.templatePane.FooterHint()
		default:
			hint = a.todosPane.FooterHint()
		}
	}

	return footerStyle.Render(counts + " · " + hint)
}

func pluralize(n int, singular string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %ss", n, singular)
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
	a.promptsPane = a.promptsPane.SetSize(a.contentWidth, a.contentHeight)
	return a
}

// handleEditorSave saves the editor content to the correct store based on the
// message's target field, refreshes the pane, and transitions to the viewer.
func (a App) handleEditorSave(m editorSaveMsg) App {
	var name string
	var err error

	if m.target == tabPrompts {
		name, err = a.promptStore.SavePrompt(m.name, m.body)
	} else {
		name, err = a.store.SaveNote(m.name, m.body)
	}
	if err != nil {
		log.Printf("wtpad: failed to save: %v", err)
		a.mode = modeNormal
		return a
	}

	if m.target == tabPrompts {
		if prompts, err := a.promptStore.ListPrompts(); err != nil {
			log.Printf("wtpad: failed to list prompts after save: %v", err)
		} else {
			a.promptsPane = a.promptsPane.SetPrompts(prompts)
		}
	} else {
		if notes, err := a.store.ListNotes(); err != nil {
			log.Printf("wtpad: failed to list notes after save: %v", err)
		} else {
			a.notesPane = a.notesPane.SetNotes(notes)
		}
	}

	a.viewerPane = a.viewerPane.openViewer(name, m.body, a.width, a.height)
	a.mode = modeViewer
	return a
}

// switchTab switches to the given tab.
func (a App) switchTab(tab activeTab) App {
	a.activeTab = tab
	a.todosPane = a.todosPane.SetFocus(tab == tabTodos)
	a.notesPane = a.notesPane.SetFocus(tab == tabNotes)
	a.promptsPane = a.promptsPane.SetFocus(tab == tabPrompts)
	return a
}
