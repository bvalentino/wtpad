package tui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// borderSize is the horizontal/vertical space consumed by a rounded border.
const borderSize = 2

type focusPane int

const (
	focusTodos focusPane = iota
	focusNotes
)

type appMode int

const (
	modeNormal appMode = iota
	modeInput
	modeEditor
	modeHelp
)

// statusBarHeight is the number of terminal rows reserved for the status bar.
const statusBarHeight = 1

type App struct {
	store      *store.Store
	width      int
	height     int
	todosWidth int
	notesWidth int
	focus      focusPane
	mode       appMode
	todosPane  todosModel
	notesPane  notesModel
	editorPane editorModel
	statusBar  statusBarModel
}

func New(s *store.Store, todos []model.Todo, notes []model.Note, dir, branch string) App {
	tp := newTodos(todos, s)
	np := newNotes(notes, s)
	sb := newStatusBar(dir, branch)
	open, done := tp.Counts()
	sb.openCount = open
	sb.doneCount = done
	return App{
		store:      s,
		focus:      focusTodos,
		mode:       modeNormal,
		todosPane:  tp,
		notesPane:  np,
		editorPane: newEditorModel(s),
		statusBar:  sb,
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case enterInputMsg:
		a.mode = modeInput
		a = a.refreshStatusBar()
		return a, nil
	case exitInputMsg:
		a.mode = modeNormal
		a = a.refreshStatusBar()
		return a, nil
	case enterEditorMsg:
		m := msg.(enterEditorMsg)
		a.editorPane = a.editorPane.openEditor(m.name, m.body, a.width, a.height)
		a.mode = modeEditor
		a = a.refreshStatusBar()
		return a, nil
	case saveNoteMsg:
		notes, err := a.store.ListNotes()
		if err != nil {
			log.Printf("wtpad: failed to list notes after save: %v", err)
		} else {
			a.notesPane = a.notesPane.SetNotes(notes)
		}
		a.mode = modeNormal
		a = a.refreshStatusBar()
		return a, nil
	case exitEditorMsg:
		a.mode = modeNormal
		a = a.refreshStatusBar()
		return a, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a = a.layoutPanes()
		if a.mode == modeEditor {
			var cmd tea.Cmd
			a.editorPane, cmd = a.editorPane.Update(msg)
			return a, cmd
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
			case "tab":
				a = a.toggleFocus()
				return a, nil
			case "q":
				return a, tea.Quit
			}
		}
	}

	// Delegate to editor when in editor mode
	if a.mode == modeEditor {
		var cmd tea.Cmd
		a.editorPane, cmd = a.editorPane.Update(msg)
		return a, cmd
	}

	// Delegate to focused pane
	var cmd tea.Cmd
	switch a.focus {
	case focusTodos:
		a.todosPane, cmd = a.todosPane.Update(msg)
	case focusNotes:
		a.notesPane, cmd = a.notesPane.Update(msg)
	}
	a = a.refreshStatusBar()
	return a, cmd
}

func (a App) View() string {
	if a.mode == modeEditor {
		return a.editorPane.View()
	}

	paneHeight := a.height - borderSize - statusBarHeight

	todosStyle := unfocusedBorder
	notesStyle := unfocusedBorder
	if a.focus == focusTodos {
		todosStyle = focusedBorder
	} else {
		notesStyle = focusedBorder
	}

	todosView := todosStyle.
		Width(a.todosWidth - borderSize).
		Height(paneHeight).
		Render(a.todosPane.View())

	notesView := notesStyle.
		Width(a.notesWidth - borderSize).
		Height(paneHeight).
		Render(a.notesPane.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, todosView, notesView)

	return lipgloss.JoinVertical(lipgloss.Left, panes, a.statusBar.View(a.width))
}

func (a App) layoutPanes() App {
	a.todosWidth = a.width * 2 / 5
	a.notesWidth = a.width - a.todosWidth

	paneHeight := a.height - borderSize - statusBarHeight
	a.todosPane = a.todosPane.SetSize(a.todosWidth-borderSize, paneHeight)
	a.notesPane = a.notesPane.SetSize(a.notesWidth-borderSize, paneHeight)
	return a
}

func (a App) refreshStatusBar() App {
	open, done := a.todosPane.Counts()
	a.statusBar.openCount = open
	a.statusBar.doneCount = done

	switch a.mode {
	case modeNormal:
		a.statusBar.hint = "? help · tab switch"
	case modeInput:
		a.statusBar.hint = "enter confirm · esc cancel"
	case modeEditor:
		a.statusBar.hint = "ctrl+s save · esc discard"
	case modeHelp:
		a.statusBar.hint = "esc close"
	}
	return a
}

func (a App) toggleFocus() App {
	if a.focus == focusTodos {
		a.focus = focusNotes
	} else {
		a.focus = focusTodos
	}
	a.todosPane = a.todosPane.SetFocus(a.focus == focusTodos)
	a.notesPane = a.notesPane.SetFocus(a.focus == focusNotes)
	return a
}
