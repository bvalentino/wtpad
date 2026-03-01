package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
)

type notesModel struct {
	notes   []model.Note
	width   int
	height  int
	focused bool
}

func newNotes(notes []model.Note) notesModel {
	return notesModel{notes: notes}
}

func (m *notesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *notesModel) SetFocus(focused bool) {
	m.focused = focused
}

func (m notesModel) Update(msg tea.Msg) (notesModel, tea.Cmd) {
	return m, nil
}

func (m notesModel) View() string {
	return "Notes"
}
