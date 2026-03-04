package tui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

type notesModel struct {
	listPane
	store *store.Store
}

func newNotes(notes []model.Note, s *store.Store) notesModel {
	m := notesModel{
		listPane: listPane{items: notesToItems(notes)},
		store:    s,
	}
	m.listPane = m.listPane.loadBodies(m.loadBodyFn())
	return m
}

func (m notesModel) SetSize(w, h int) notesModel {
	m.listPane = m.listPane.setSize(w, h)
	return m
}

func (m notesModel) SetFocus(focused bool) notesModel {
	m.listPane = m.listPane.setFocus(focused)
	return m
}

func (m notesModel) Update(msg tea.Msg) (notesModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// Handle "y" for delete confirmation — perform the actual delete
	if m.confirmDelete && keyMsg.String() == "y" {
		m = m.deleteSelected()
		m.confirmDelete = false
		return m, nil
	}

	var cmd tea.Cmd
	var handled bool
	m.listPane, cmd, handled = m.listPane.handleKey(keyMsg)
	if handled {
		return m, cmd
	}

	return m, nil
}

func (m notesModel) View() string {
	if len(m.items) == 0 {
		return "No notes yet. Press 'a' to create one."
	}

	var barContent string
	if m.confirmDelete {
		barContent = listConfirm.Render("Delete note? (y to confirm)")
	} else {
		barContent = hintStyle.Render("Add Note (a)")
	}

	return assembleListView(m.listPane, barContent)
}

func (m notesModel) loadBodyFn() func(string) (string, error) {
	if m.store == nil {
		return nil
	}
	return func(name string) (string, error) {
		loaded, err := m.store.LoadNote(name)
		if err != nil {
			return "", err
		}
		return loaded.Body, nil
	}
}

func (m notesModel) deleteSelected() notesModel {
	if len(m.items) == 0 {
		return m
	}
	name := m.items[m.cursor].Name
	if err := m.store.DeleteNote(name); err != nil {
		log.Printf("wtpad: failed to delete note %s: %v", name, err)
		return m
	}
	m.listPane = m.listPane.removeItem(m.cursor)
	return m
}

// SetNotes replaces the notes slice (used after editor saves a new/updated note).
func (m notesModel) SetNotes(notes []model.Note) notesModel {
	m.listPane = m.listPane.setItems(notes, m.loadBodyFn())
	return m
}

// Init satisfies the tea.Model interface for standalone use.
func (m notesModel) Init() tea.Cmd {
	return nil
}

// SelectedNote returns the currently selected note, or nil if empty.
func (m notesModel) SelectedNote() *model.Note {
	item := m.selectedItem()
	if item == nil {
		return nil
	}
	return &model.Note{
		Name:      item.Name,
		Body:      item.Body,
		CreatedAt: item.CreatedAt,
	}
}

