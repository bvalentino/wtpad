package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
)

type todosModel struct {
	todos   []model.Todo
	width   int
	height  int
	focused bool
}

func newTodos(todos []model.Todo) todosModel {
	return todosModel{todos: todos, focused: true}
}

func (m *todosModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *todosModel) SetFocus(focused bool) {
	m.focused = focused
}

func (m todosModel) Update(msg tea.Msg) (todosModel, tea.Cmd) {
	return m, nil
}

func (m todosModel) View() string {
	return "Todos"
}
