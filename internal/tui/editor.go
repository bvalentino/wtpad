package tui

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/store"
)

const noteTimeFmt = "20060102-150405"

// saveNoteMsg signals root that the editor saved a note successfully.
type saveNoteMsg struct{ name string }

// exitEditorMsg signals root to leave editor mode without saving.
type exitEditorMsg struct{}

type editorModel struct {
	textarea       textarea.Model
	store          *store.Store
	name           string // empty = new note, non-empty = editing existing
	initialBody    string // snapshot for dirty detection
	width          int
	height         int
	confirmDiscard bool
	err            error
}

func newEditorModel(s *store.Store) editorModel {
	ta := textarea.New()
	ta.Placeholder = "Start writing..."
	ta.CharLimit = 0 // no limit
	ta.ShowLineNumbers = false
	return editorModel{
		textarea: ta,
		store:    s,
	}
}

// openEditor prepares the editor for a new or existing note.
// w and h are the content area dimensions (already excluding app chrome).
func (e editorModel) openEditor(name, body string, w, h int) editorModel {
	e.name = name
	e.initialBody = body
	e.confirmDiscard = false
	e.err = nil
	e.textarea.SetValue(body)
	e.textarea.Focus()
	return e.resize(w, h)
}

// resize updates the editor dimensions to fit within the content area.
func (e editorModel) resize(w, h int) editorModel {
	e.width = w
	e.height = h
	e.textarea.SetWidth(w)
	taHeight := h - 1 // minus header line
	if taHeight < 1 {
		taHeight = 1
	}
	e.textarea.SetHeight(taHeight)
	return e
}

func (e editorModel) Update(msg tea.Msg) (editorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Confirm-discard mode: only y/n matter
		if e.confirmDiscard {
			switch msg.String() {
			case "y":
				return e, func() tea.Msg { return exitEditorMsg{} }
			default:
				e.confirmDiscard = false
				return e, nil
			}
		}

		switch msg.String() {
		case "ctrl+s", "ctrl+d":
			return e.save()
		case "esc":
			if e.dirty() {
				e.confirmDiscard = true
				return e, nil
			}
			return e, func() tea.Msg { return exitEditorMsg{} }
		}
	}

	// Delegate all other messages to textarea
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

func (e editorModel) save() (editorModel, tea.Cmd) {
	body := e.textarea.Value()
	name, err := e.store.SaveNote(e.name, body)
	if err != nil {
		log.Printf("wtpad: failed to save note: %v", err)
		e.err = err
		return e, nil
	}
	return e, func() tea.Msg { return saveNoteMsg{name: name} }
}

func (e editorModel) dirty() bool {
	return e.textarea.Value() != e.initialBody
}

// FooterHint returns the contextual hint for the app footer.
func (e editorModel) FooterHint() string {
	switch {
	case e.err != nil:
		return fmt.Sprintf("save failed: %v", e.err)
	case e.confirmDiscard:
		return "discard changes? y/n"
	default:
		return "ctrl+s save · esc discard"
	}
}

func (e editorModel) View() string {
	header := "New Note"
	if e.name != "" {
		if t, err := time.Parse(noteTimeFmt, e.name); err == nil {
			header = fmt.Sprintf("Editing — %s", t.Format("Jan 02 15:04"))
		} else {
			header = fmt.Sprintf("Editing — %s", e.name)
		}
	}
	headerLine := editorHeader.Render(header)

	return headerLine + "\n" + e.textarea.View()
}
