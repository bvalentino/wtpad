package tui

import (
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
}

func New(s *store.Store, todos []model.Todo, notes []model.Note) App {
	tp := newTodos(todos, s)
	np := newNotes(notes)
	return App{
		store:     s,
		focus:     focusTodos,
		mode:      modeNormal,
		todosPane: tp,
		notesPane: np,
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
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a = a.layoutPanes()
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

	// Delegate to focused pane
	var cmd tea.Cmd
	switch a.focus {
	case focusTodos:
		a.todosPane, cmd = a.todosPane.Update(msg)
	case focusNotes:
		a.notesPane, cmd = a.notesPane.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	todosStyle := unfocusedBorder
	notesStyle := unfocusedBorder
	if a.focus == focusTodos {
		todosStyle = focusedBorder
	} else {
		notesStyle = focusedBorder
	}

	todosView := todosStyle.
		Width(a.todosWidth - borderSize).
		Height(a.height - borderSize).
		Render(a.todosPane.View())

	notesView := notesStyle.
		Width(a.notesWidth - borderSize).
		Height(a.height - borderSize).
		Render(a.notesPane.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, todosView, notesView)
}

func (a App) layoutPanes() App {
	a.todosWidth = a.width * 2 / 5
	a.notesWidth = a.width - a.todosWidth

	a.todosPane = a.todosPane.SetSize(a.todosWidth-borderSize, a.height-borderSize)
	a.notesPane = a.notesPane.SetSize(a.notesWidth-borderSize, a.height-borderSize)
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
