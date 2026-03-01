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

func (m notesModel) SetSize(w, h int) notesModel {
	m.width = w
	m.height = h
	return m
}

func (m notesModel) SetFocus(focused bool) notesModel {
	m.focused = focused
	return m
}

func (m notesModel) Update(msg tea.Msg) (notesModel, tea.Cmd) {
	return m, nil
}

func (m notesModel) View() string {
	return "Notes"
}
