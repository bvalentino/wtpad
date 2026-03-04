package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// editorSaveMsg signals root that the editor wants to save.
// Root dispatches to the correct store based on the target field.
type editorSaveMsg struct {
	name   string // original name (empty for new)
	body   string
	target activeTab // which tab originated the edit
}

// exitEditorMsg signals root to leave editor mode without saving.
type exitEditorMsg struct{}

type editorModel struct {
	textarea       textarea.Model
	name           string // empty = new note, non-empty = editing existing
	initialBody    string // snapshot for dirty detection
	entityName     string // "Note" or "Prompt" — used in overlay title
	target         activeTab
	width          int
	height         int
	confirmDiscard bool
}

func newEditorModel() editorModel {
	ta := textarea.New()
	ta.Placeholder = "Start writing..."
	ta.CharLimit = 0 // no limit
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	return editorModel{textarea: ta}
}

// openEditor prepares the editor for a new or existing item.
func (e editorModel) openEditor(name, body, entityName string, target activeTab, w, h int) editorModel {
	e.name = name
	e.initialBody = body
	e.entityName = entityName
	e.target = target
	e.confirmDiscard = false
	e.textarea.SetValue(body)
	e.textarea.Focus()
	return e.resize(w, h)
}

// resize updates the editor dimensions accounting for the overlay box.
func (e editorModel) resize(w, h int) editorModel {
	e.width = w
	e.height = h
	e.textarea.SetWidth(overlayContentWidth(w))
	e.textarea.SetHeight(overlayContentHeight(h))
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
	name := e.name
	body := e.textarea.Value()
	target := e.target
	return e, func() tea.Msg { return editorSaveMsg{name: name, body: body, target: target} }
}

func (e editorModel) dirty() bool {
	return e.textarea.Value() != e.initialBody
}

// FooterHint returns the contextual hint for the editor overlay.
func (e editorModel) FooterHint() string {
	switch {
	case e.confirmDiscard:
		return "discard changes? y/n"
	default:
		return "ctrl+s save · esc discard"
	}
}

func (e editorModel) View() string {
	if e.width == 0 || e.height == 0 {
		return ""
	}

	taLines := strings.Split(e.textarea.View(), "\n")
	title := "Edit " + e.entityName
	if e.name == "" {
		title = "New " + e.entityName
	}
	return renderOverlayBox(title, taLines, e.width, e.height, e.FooterHint())
}
