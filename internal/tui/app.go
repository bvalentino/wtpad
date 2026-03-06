package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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
	tabAI
)

type appMode int

const (
	modeNormal appMode = iota
	modeInput
	modeEditor
	modeViewer
	modeHelp
	modeTemplate
	modeTitleInput
)

// Layout constants
const (
	tabStripHeight    = 3
	footerHeight      = 1
	sideBorderSize    = 2  // left + right │ borders
	maxTitleLineWidth = 35 // max chars per line inside the title box
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
	store       *store.Store
	aiEvents    <-chan aiFileChangedMsg    // long-lived watcher channel; nil if watcher failed
	titleEvents <-chan titleFileChangedMsg // long-lived watcher channel; nil if watcher failed
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
	title  string
	branch string

	titleInput textinput.Model

	todosPane    todosModel
	notesPane    notesModel
	promptsPane  promptsModel
	aiPane       aiModel
	editorPane   editorModel
	viewerPane   viewerModel
	helpPane     helpModel
	templatePane templateModal

	promptStore *store.PromptStore
}

// AppConfig holds the parameters for creating a new App.
type AppConfig struct {
	Store         *store.Store
	TemplateStore *store.TemplateStore
	PromptStore   *store.PromptStore
	Todos         []model.Todo
	Notes         []model.Note
	Prompts       []model.Note
	AITodos       []model.Todo
	Branch        string
	Title         string
}

func New(cfg AppConfig) App {
	tp := newTodos(cfg.Todos, cfg.Store)
	np := newNotes(cfg.Notes, cfg.Store)
	pp := newPrompts(cfg.Prompts, cfg.PromptStore)
	ti := textinput.New()
	ti.Prompt = "Title: "
	title := cfg.Title
	wch := startDirWatcher(cfg.Store)
	return App{
		store:        cfg.Store,
		aiEvents:     wch.ai,
		titleEvents:  wch.title,
		promptStore:  cfg.PromptStore,
		activeTab:    tabTodos,
		mode:         modeNormal,
		title:        title,
		branch:       cfg.Branch,
		titleInput:   ti,
		todosPane:    tp,
		notesPane:    np,
		promptsPane:  pp,
		aiPane:       newAI(cfg.AITodos, cfg.Store),
		editorPane:   newEditorModel(),
		templatePane: newTemplateModal(cfg.TemplateStore),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		waitForChange(a.aiEvents),
		waitForChange(a.titleEvents),
		tea.SetWindowTitle(a.windowTitle()),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case clearAIStatusMsg:
		a.aiPane, _ = a.aiPane.Update(msg)
		return a, nil
	case aiFileChangedMsg:
		a.aiPane, _ = a.aiPane.Update(msg)
		if a.activeTab == tabAI && !a.showAITab() {
			a = a.switchTab(tabTodos)
		}
		// Re-subscribe for the next event from the long-lived watcher.
		return a, waitForChange(a.aiEvents)
	case titleFileChangedMsg:
		var cmds []tea.Cmd
		if title, err := a.store.LoadTitle(); err != nil {
			log.Printf("wtpad: failed to reload title: %v", err)
		} else {
			if title != a.title {
				a.title = title
				cmds = append(cmds, tea.SetWindowTitle(a.windowTitle()))
			}
		}
		cmds = append(cmds, waitForChange(a.titleEvents))
		return a, tea.Batch(cmds...)
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
		a.helpPane = a.helpPane.open(a.width, a.height)
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
					if a.showAITab() {
						a = a.switchTab(tabAI)
					} else {
						a = a.switchTab(tabTodos)
					}
				case tabAI:
					a = a.switchTab(tabTodos)
				}
				return a, nil
			case "?":
				return a, func() tea.Msg { return enterHelpMsg{} }
			case "t":
				a.titleInput.SetValue(a.title)
				a.titleInput.Focus()
				a.mode = modeTitleInput
				return a, textinput.Blink
			case "q":
				return a, tea.Quit
			}
		}

		// Handle title input mode
		if a.mode == modeTitleInput {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				title := strings.TrimSpace(a.titleInput.Value())
				if err := a.store.SaveTitle(title); err != nil {
					log.Printf("wtpad: failed to save title: %v", err)
				} else {
					a.title = title
				}
				a.titleInput.Blur()
				a.mode = modeNormal
				a = a.layoutVertical()
				cmd = tea.SetWindowTitle(a.windowTitle())
			case "esc":
				a.titleInput.Blur()
				a.mode = modeNormal
			default:
				a.titleInput, _ = a.titleInput.Update(msg)
			}
			return a, cmd
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
	case tabAI:
		a.aiPane, cmd = a.aiPane.Update(msg)
		if !a.showAITab() {
			a = a.switchTab(tabTodos)
		}
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
// When a title is set, a ┏━━┓ box is overlaid on lines 1–3 of the logo (full mode)
// or the title is right-aligned on the compact line.
func (a App) renderHeader() string {
	if a.showFullHeader {
		lines := strings.Split(asciiHeader, "\n")

		// Center the ASCII art within the terminal width.
		logoWidth := 0
		for _, l := range lines {
			if w := len(l); w > logoWidth {
				logoWidth = w
			}
		}
		if pad := (a.width - logoWidth) / 2; pad > 0 {
			prefix := strings.Repeat(" ", pad)
			for i, l := range lines {
				lines[i] = prefix + l
			}
		}

		if a.title != "" && a.width > 4 {
			titleLines := wrapTitle(a.title, maxTitleLineWidth)
			if len(titleLines) > 3 {
				last := titleLines[2]
				if lipgloss.Width(last)+1 <= maxTitleLineWidth {
					last += "…"
				} else {
					// Replace last character with ellipsis
					runes := []rune(last)
					last = string(runes[:len(runes)-1]) + "…"
				}
				titleLines = append(titleLines[:2], last)
			}
			numContent := len(titleLines)

			// Box width based on longest wrapped line.
			maxLineW := 0
			for _, l := range titleLines {
				if w := lipgloss.Width(l); w > maxLineW {
					maxLineW = w
				}
			}
			boxInner := maxLineW + 8 // 4-char padding each side
			boxWidth := boxInner + 2
			if boxWidth > a.width-2 {
				boxWidth = a.width - 2
				boxInner = boxWidth - 2
			}
			if boxInner < maxLineW {
				boxInner = maxLineW
				boxWidth = boxInner + 2
			}
			boxStart := (a.width - boxWidth) / 2
			if boxStart < 0 {
				boxStart = 0
			}
			boxEnd := boxStart + boxWidth

			// Pad lines so slicing up to boxEnd is safe.
			for i := range lines {
				if len(lines[i]) < boxEnd {
					lines[i] += strings.Repeat(" ", boxEnd-len(lines[i]))
				}
			}

			// Box occupies lines topIdx..bottomIdx of the ASCII art.
			topIdx := 1
			bottomIdx := topIdx + numContent + 1

			result := make([]string, len(lines))
			for i, line := range lines {
				left := line[:boxStart]
				right := ""
				if len(line) > boxEnd {
					right = line[boxEnd:]
				}

				switch {
				case i == topIdx: // ┏━━━┓
					result[i] = headerStyle.Render(left) +
						titleStyle.Render("┏"+strings.Repeat("━", boxInner)+"┓") +
						headerStyle.Render(right)
				case i > topIdx && i < bottomIdx: // ┃ title ┃
					tl := titleLines[i-topIdx-1]
					tlW := lipgloss.Width(tl)
					lp := (boxInner - tlW) / 2
					rp := boxInner - tlW - lp
					result[i] = headerStyle.Render(left) +
						titleStyle.Render("┃"+strings.Repeat(" ", lp)+tl+strings.Repeat(" ", rp)+"┃") +
						headerStyle.Render(right)
				case i == bottomIdx: // ┗━━━┛
					result[i] = headerStyle.Render(left) +
						titleStyle.Render("┗"+strings.Repeat("━", boxInner)+"┛") +
						headerStyle.Render(right)
				default:
					result[i] = headerStyle.Render(line)
				}
			}
			return strings.Join(result, "\n")
		}
		return headerStyle.Render(strings.Join(lines, "\n"))
	}

	compact := "wtpad"
	if a.branch != "" {
		compact += " · " + a.branch
	}
	left := headerStyle.Render(compact)
	if a.title != "" {
		styledTitle := titleStyle.Render(a.title)
		titleW := lipgloss.Width(styledTitle)
		leftW := lipgloss.Width(left)
		gap := a.width - leftW - titleW
		if gap < 1 {
			gap = 1
		}
		return left + strings.Repeat(" ", gap) + styledTitle
	}
	return left
}

// renderTabStrip returns the 3-line tab chrome.
// Lines 1-2 (top border + label) are rendered by lipgloss borders with a
// 1-space gap between tabs. Line 3 (connection to content area) is manually
// constructed so the active tab opening flows into the content box.
//
// When showAITab() is true, a 4th "AI" tab is appended.
func (a App) renderTabStrip() string {
	renderTab := func(label string, active bool) string {
		if active {
			return activeTabStyle.Render(label)
		}
		return inactiveTabStyle.Render(label)
	}

	todoTab := renderTab("Todo", a.activeTab == tabTodos)
	noteTab := renderTab("Notes", a.activeTab == tabNotes)
	promptTab := renderTab("Prompts", a.activeTab == tabPrompts)

	showAI := a.showAITab()

	// Lines 1-2: join tabs with a 1-space gap
	gap := lipgloss.NewStyle().Width(1)
	var row string
	if showAI {
		aiTab := renderTab("AI", a.activeTab == tabAI)
		row = lipgloss.JoinHorizontal(lipgloss.Top,
			todoTab, gap.Render(" "), noteTab, gap.Render(" "), promptTab, gap.Render(" "), aiTab,
		)
	} else {
		row = lipgloss.JoinHorizontal(lipgloss.Top,
			todoTab, gap.Render(" "), noteTab, gap.Render(" "), promptTab,
		)
	}

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

	var line3 string

	if showAI {
		aiTab := renderTab("AI", a.activeTab == tabAI)
		aiW := lipgloss.Width(aiTab)
		aiInner := aiW - 2

		// tabsSpan = total columns consumed by 4 tabs + 3 gaps (1 char each)
		// = todoW(1) + gap(1) + noteW(1) + gap(1) + promptW(1) + gap(1) + aiW(1)
		tabsSpan := todoW + 1 + noteW + 1 + promptW + 1 + aiW

		// fill = remaining space minus ╮(1) right-edge cap
		fill := a.width - tabsSpan - 1
		if fill < 0 {
			fill = 0
		}

		// Character count per line3 case (4 tabs):
		// Fixed chars: left(1) + 3 junctions of "X─┴"(3 each=9) + ┴(1) + ╮(1) = 12
		// Variable: todoInner + noteInner + promptInner + aiInner + fill
		// Total: 12 + todoInner + noteInner + promptInner + aiInner + fill = width ✓
		switch a.activeTab {
		case tabTodos:
			// │<spaces>╰─┴<─>┴─┴<─>┴─┴<─>┴<─>╮
			line3 = "│" + strings.Repeat(" ", todoInner) +
				"╰─┴" + strings.Repeat("─", noteInner) +
				"┴─┴" + strings.Repeat("─", promptInner) +
				"┴─┴" + strings.Repeat("─", aiInner) +
				"┴" + strings.Repeat("─", fill) + "╮"
		case tabNotes:
			// ├<─>┴─╯<spaces>╰─┴<─>┴─┴<─>┴<─>╮
			line3 = "├" + strings.Repeat("─", todoInner) +
				"┴─╯" + strings.Repeat(" ", noteInner) +
				"╰─┴" + strings.Repeat("─", promptInner) +
				"┴─┴" + strings.Repeat("─", aiInner) +
				"┴" + strings.Repeat("─", fill) + "╮"
		case tabPrompts:
			// ├<─>┴─┴<─>┴─╯<spaces>╰─┴<─>┴<─>╮
			line3 = "├" + strings.Repeat("─", todoInner) +
				"┴─┴" + strings.Repeat("─", noteInner) +
				"┴─╯" + strings.Repeat(" ", promptInner) +
				"╰─┴" + strings.Repeat("─", aiInner) +
				"┴" + strings.Repeat("─", fill) + "╮"
		case tabAI:
			// ├<─>┴─┴<─>┴─┴<─>┴─╯<spaces>╰<─>╮
			line3 = "├" + strings.Repeat("─", todoInner) +
				"┴─┴" + strings.Repeat("─", noteInner) +
				"┴─┴" + strings.Repeat("─", promptInner) +
				"┴─╯" + strings.Repeat(" ", aiInner) +
				"╰" + strings.Repeat("─", fill) + "╮"
		}
	} else {
		// 3-tab layout (no AI tab)
		// tabsSpan = total columns consumed by 3 tabs + 2 gaps (1 char each)
		tabsSpan := todoW + 1 + noteW + 1 + promptW

		// fill = remaining space minus ╮(1) right-edge cap
		fill := a.width - tabsSpan - 1
		if fill < 0 {
			fill = 0
		}

		// Character count per line3 case (3 tabs):
		// Fixed chars: left(1) + 2 junctions of "X─┴"(3 each=6) + ┴(1) + ╮(1) = 9
		// Variable: todoInner + noteInner + promptInner + fill
		// Total: 9 + todoInner + noteInner + promptInner + fill = width ✓
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
	case tabAI:
		content = a.aiPane.View()
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
	// Mode-level overrides take precedence over tab-level hints.
	switch a.mode {
	case modeHelp:
		return footerStyle.Render(a.helpPane.FooterHint())
	case modeEditor:
		return footerStyle.Render(a.editorPane.FooterHint())
	case modeViewer:
		return footerStyle.Render(a.viewerPane.FooterHint())
	case modeTitleInput:
		return footerStyle.Render(a.titleInput.View())
	}

	var counts, hint string

	switch a.activeTab {
	case tabAI:
		counts = pluralize(a.aiPane.count(), "task")
		if msg := a.aiPane.StatusMsg(); msg != "" {
			return footerStyle.Render(counts + " · " + msg)
		}
		hint = a.aiPane.FooterHint()

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

	a.titleInput.Width = a.width - 10
	a.todosPane = a.todosPane.SetSize(a.contentWidth, a.contentHeight)
	a.notesPane = a.notesPane.SetSize(a.contentWidth, a.contentHeight)
	a.promptsPane = a.promptsPane.SetSize(a.contentWidth, a.contentHeight)
	a.aiPane = a.aiPane.SetSize(a.contentWidth, a.contentHeight)
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

// wrapTitle splits a title into lines no wider than width, breaking on spaces.
func wrapTitle(title string, width int) []string {
	words := strings.Fields(title)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if lipgloss.Width(current)+1+lipgloss.Width(word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	return append(lines, current)
}

// switchTab switches to the given tab.
func (a App) switchTab(tab activeTab) App {
	a.activeTab = tab
	a.todosPane = a.todosPane.SetFocus(tab == tabTodos)
	a.notesPane = a.notesPane.SetFocus(tab == tabNotes)
	a.promptsPane = a.promptsPane.SetFocus(tab == tabPrompts)
	a.aiPane = a.aiPane.SetFocus(tab == tabAI)
	return a
}

// windowTitle returns the terminal window title to set via OSC 2.
// Control characters are stripped to prevent terminal escape injection.
func (a App) windowTitle() string {
	if a.title == "" {
		return "wtpad"
	}
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, a.title)
}

// showAITab reports whether the AI tab should be visible.
func (a App) showAITab() bool {
	return a.aiPane.HasItems() || a.aiPane.fileExists
}
