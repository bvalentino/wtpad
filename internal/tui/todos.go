package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// Mode transition messages — handled by root App to set appMode.
type enterInputMsg struct{}
type exitInputMsg struct{}

type todosModel struct {
	todos        []model.Todo
	store        *store.Store
	cursor       int
	scrollOffset int
	width        int
	height       int
	focused      bool
	inputActive  bool
	input        textinput.Model
	editIndex    int // -1 = adding new, >= 0 = editing existing
}

func newTodos(todos []model.Todo, s *store.Store) todosModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 256

	m := todosModel{
		todos:     sortTodos(todos),
		store:     s,
		focused:   true,
		input:     ti,
		editIndex: -1,
	}
	return m
}

func (m todosModel) SetSize(w, h int) todosModel {
	m.width = w
	m.height = h
	m.input.Width = w - 4 // leave room for prompt and padding
	return m
}

func (m todosModel) SetFocus(focused bool) todosModel {
	m.focused = focused
	return m
}

func (m todosModel) Update(msg tea.Msg) (todosModel, tea.Cmd) {
	if m.inputActive {
		return m.updateInput(msg)
	}
	return m.updateNormal(msg)
}

func (m todosModel) updateNormal(msg tea.Msg) (todosModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		m = m.moveCursor(1)
	case "k", "up":
		m = m.moveCursor(-1)
	case "a":
		m.input.SetValue("")
		m.input.Focus()
		m.inputActive = true
		m.editIndex = -1
		return m, func() tea.Msg { return enterInputMsg{} }
	case "enter":
		if len(m.todos) > 0 {
			m.input.SetValue(m.todos[m.cursor].Text)
			m.input.Focus()
			m.inputActive = true
			m.editIndex = m.cursor
			return m, func() tea.Msg { return enterInputMsg{} }
		}
	case "d", " ":
		m = m.toggleDone()
	case "x", "delete":
		m = m.deleteCurrent()
	case "D":
		m = m.purgeDone()
	}

	return m, nil
}

func (m todosModel) updateInput(msg tea.Msg) (todosModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				if m.editIndex >= 0 && m.editIndex < len(m.todos) {
					m.todos[m.editIndex].Text = text
				} else {
					m.todos = append(m.todos, model.Todo{Text: text})
					m.todos = sortTodos(m.todos)
					m = m.clampCursor()
					m = m.adjustScroll()
				}
				m.save()
			}
			m.inputActive = false
			m.input.Blur()
			return m, func() tea.Msg { return exitInputMsg{} }
		case "esc":
			m.inputActive = false
			m.input.Blur()
			return m, func() tea.Msg { return exitInputMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m todosModel) View() string {
	if len(m.todos) == 0 && !m.inputActive {
		return "No todos yet. Press 'a' to add one."
	}

	var b strings.Builder

	// Available lines for the todo list (reserve 1 for input if active)
	visibleLines := m.height
	if m.inputActive {
		visibleLines--
	}
	if visibleLines < 1 {
		visibleLines = 1
	}

	end := m.scrollOffset + visibleLines
	if end > len(m.todos) {
		end = len(m.todos)
	}

	for i := m.scrollOffset; i < end; i++ {
		todo := m.todos[i]
		var prefix string
		if todo.Done {
			prefix = "✓ "
		} else {
			prefix = "○ "
		}

		line := fmt.Sprintf("%s%s", prefix, todo.Text)
		line = truncate(line, m.width)

		selected := i == m.cursor && m.focused
		switch {
		case todo.Done && selected:
			line = todoDoneSelected.Render(line)
		case todo.Done:
			line = todoDone.Render(line)
		case selected:
			line = todoSelected.Render(line)
		}

		b.WriteString(line)
		if i < end-1 || m.inputActive {
			b.WriteString("\n")
		}
	}

	if m.inputActive {
		b.WriteString(m.input.View())
	}

	return b.String()
}

// moveCursor moves the cursor by delta, clamps, and adjusts scroll.
func (m todosModel) moveCursor(delta int) todosModel {
	m.cursor += delta
	m = m.clampCursor()
	m = m.adjustScroll()
	return m
}

// clampCursor ensures cursor is within [0, len(todos)-1].
func (m todosModel) clampCursor() todosModel {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if max := len(m.todos) - 1; m.cursor > max {
		if max < 0 {
			m.cursor = 0
		} else {
			m.cursor = max
		}
	}
	return m
}

// adjustScroll ensures scrollOffset keeps the cursor visible within the pane.
func (m todosModel) adjustScroll() todosModel {
	visibleLines := m.height
	if m.inputActive {
		visibleLines--
	}
	if visibleLines < 1 {
		visibleLines = 1
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visibleLines {
		m.scrollOffset = m.cursor - visibleLines + 1
	}
	return m
}

// toggleDone toggles the Done state of the selected todo, re-sorts, and saves.
func (m todosModel) toggleDone() todosModel {
	if len(m.todos) == 0 {
		return m
	}
	m.todos[m.cursor].Done = !m.todos[m.cursor].Done
	m.todos = sortTodos(m.todos)
	m = m.clampCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// deleteCurrent removes the selected todo and saves.
func (m todosModel) deleteCurrent() todosModel {
	if len(m.todos) == 0 {
		return m
	}
	m.todos = append(m.todos[:m.cursor], m.todos[m.cursor+1:]...)
	m = m.clampCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// purgeDone removes all completed todos and saves.
func (m todosModel) purgeDone() todosModel {
	filtered := make([]model.Todo, 0, len(m.todos))
	for _, t := range m.todos {
		if !t.Done {
			filtered = append(filtered, t)
		}
	}
	m.todos = filtered
	m = m.clampCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// save persists todos to disk, logging on failure.
func (m todosModel) save() {
	if err := m.store.SaveTodos(m.todos); err != nil {
		log.Printf("wtpad: failed to save todos: %v", err)
	}
}

// sortTodos returns todos with open items first, then done items,
// preserving relative order within each group.
func sortTodos(todos []model.Todo) []model.Todo {
	open := make([]model.Todo, 0, len(todos))
	done := make([]model.Todo, 0)
	for _, t := range todos {
		if t.Done {
			done = append(done, t)
		} else {
			open = append(open, t)
		}
	}
	return append(open, done...)
}

// truncate shortens s to fit within width.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 1 {
		return string(r[:width])
	}
	return string(r[:width-1]) + "…"
}

// Init satisfies the tea.Model interface for standalone use.
func (m todosModel) Init() tea.Cmd {
	return textinput.Blink
}

// Focused returns whether the pane is focused (used by lipgloss rendering).
func (m todosModel) Focused() bool {
	return m.focused
}

// Counts returns the number of open and done todos.
func (m todosModel) Counts() (open, done int) {
	for _, t := range m.todos {
		if t.Done {
			done++
		} else {
			open++
		}
	}
	return open, done
}
