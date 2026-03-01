package tui

import (
	"log"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// clearStatusMsg clears the transient status message after a delay.
type clearStatusMsg struct{}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// Mode transition messages — handled by root App to set appMode.
type enterInputMsg struct{}
type exitInputMsg struct{}

// todoPrefixWidth is the display width of the checkbox prefix ("○ " / "✓ ").
var todoPrefixWidth = lipgloss.Width("○ ")

type todosModel struct {
	todos        []model.Todo
	store        *store.Store
	cursor       int
	scrollOffset int
	width        int
	height       int
	textWidth    int // available columns for todo text (width minus prefix)
	focused      bool
	inputActive  bool
	input        textinput.Model
	editIndex    int // -1 = adding new, >= 0 = editing existing
	statusMsg    string
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
	m.textWidth = w - todoPrefixWidth
	if m.textWidth < 1 {
		m.textWidth = 1
	}
	m.input.Width = w - 4 // leave room for prompt and padding
	m = m.adjustScroll()
	return m
}

func (m todosModel) SetFocus(focused bool) todosModel {
	m.focused = focused
	return m
}

func (m todosModel) Update(msg tea.Msg) (todosModel, tea.Cmd) {
	if _, ok := msg.(clearStatusMsg); ok {
		m.statusMsg = ""
		return m, nil
	}
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
	case "c":
		if len(m.todos) > 0 {
			if err := clipboard.WriteAll(m.todos[m.cursor].Text); err != nil {
				m.statusMsg = "Copy failed"
			} else {
				m.statusMsg = "Copied!"
			}
			return m, clearStatusAfter(2 * time.Second)
		}
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
	linesUsed := 0
	visibleLines := m.height
	if m.inputActive {
		visibleLines--
	}
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Find where the done section starts (sortTodos guarantees open first).
	doneStart := len(m.todos)
	for i, t := range m.todos {
		if t.Done {
			doneStart = i
			break
		}
	}

	// Track whether we've rendered the hint + divider between open and done.
	hintRendered := false
	prevWasOpen := false
	indent := strings.Repeat(" ", todoPrefixWidth)

	for i := m.scrollOffset; i < len(m.todos) && linesUsed < visibleLines; i++ {
		todo := m.todos[i]

		// Insert Add hint and divider at the open→done boundary.
		if todo.Done && !hintRendered {
			hintRendered = true
			// Add (a) hint
			if linesUsed > 0 && linesUsed < visibleLines {
				b.WriteString("\n")
				linesUsed++
			}
			if linesUsed < visibleLines {
				b.WriteString("\n")
				b.WriteString(hintStyle.Render("Add TODO (a)"))
				linesUsed++
			}
			// End hint line (terminator, no count)
			if linesUsed < visibleLines {
				b.WriteString("\n")
			}
			// Blank line after hint
			if linesUsed < visibleLines {
				b.WriteString("\n")
				linesUsed++
			}
			// Divider
			if linesUsed < visibleLines {
				b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)))
				linesUsed++
			}
		}

		if linesUsed >= visibleLines {
			break
		}

		// Blank line between open items for breathing room.
		if !todo.Done && prevWasOpen && linesUsed < visibleLines {
			if linesUsed > 0 {
				b.WriteString("\n")
				linesUsed++
			}
		}

		// Newline before the item (except first rendered line).
		// This \n is a line terminator, not a visible line — don't increment linesUsed.
		if linesUsed > 0 {
			b.WriteString("\n")
		}

		// Render the todo line(s) with wrapping.
		var prefix string
		if todo.Done {
			prefix = "✓ "
		} else {
			prefix = "○ "
		}

		wrapped := wrapText(todo.Text, m.textWidth)
		selected := i == m.cursor && m.focused

		// Pick style once — same for all wrapped lines of this item.
		var style lipgloss.Style
		styled := false
		switch {
		case todo.Done && selected:
			style = todoDoneSelected
			styled = true
		case todo.Done:
			style = todoDone
			styled = true
		case selected:
			style = todoSelected
			styled = true
		}

		for li, wl := range wrapped {
			if linesUsed >= visibleLines {
				break
			}
			var line string
			if li == 0 {
				line = prefix + wl
			} else {
				// Continuation lines: newline + indent
				b.WriteString("\n")
				line = indent + wl
			}

			if styled {
				line = style.Render(line)
			}

			b.WriteString(line)
			linesUsed++
		}
		prevWasOpen = !todo.Done
	}

	// If all visible items were open, still show the Add hint at the end.
	if !hintRendered && doneStart > 0 && linesUsed > 0 && linesUsed < visibleLines {
		b.WriteString("\n")
		linesUsed++
		if linesUsed < visibleLines {
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("Add TODO (a)"))
			linesUsed++
		}
		if linesUsed < visibleLines {
			b.WriteString("\n")
			linesUsed++
		}
	}

	if m.inputActive {
		b.WriteString("\n")
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

// availableLines returns the number of lines available for todo rendering.
func (m todosModel) availableLines() int {
	h := m.height
	if m.inputActive {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// linesUpTo counts rendered lines from scrollOffset through targetIdx,
// mirroring the View() line-accounting logic (blank lines, hint/divider,
// wrapped item heights). Must stay in sync with View() — any change to
// spacing or wrapping in View() must be reflected here.
func (m todosModel) linesUpTo(targetIdx int) int {
	linesUsed := 0
	prevWasOpen := false
	hintRendered := false

	for i := m.scrollOffset; i < len(m.todos) && i <= targetIdx; i++ {
		todo := m.todos[i]

		// Hint + divider at the open→done boundary.
		if todo.Done && !hintRendered {
			hintRendered = true
			if linesUsed > 0 {
				linesUsed++ // blank before hint
			}
			linesUsed++ // hint text
			// hint line terminator (no count)
			linesUsed++ // blank after hint
			linesUsed++ // divider
		}

		// Blank line between consecutive open items.
		if !todo.Done && prevWasOpen && linesUsed > 0 {
			linesUsed++
		}

		// Newline before item is a terminator (no count).

		// The item itself — may occupy multiple lines when wrapped.
		linesUsed += len(wrapText(todo.Text, m.textWidth))
		prevWasOpen = !todo.Done
	}

	return linesUsed
}

// adjustScroll ensures scrollOffset keeps the cursor visible within the pane.
// Uses line counting to account for blank lines and hint/divider sections.
func (m todosModel) adjustScroll() todosModel {
	if m.height < 1 || len(m.todos) == 0 {
		return m
	}

	// Scroll up if cursor is above viewport.
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}

	// Scroll down if cursor is below viewport.
	avail := m.availableLines()
	for m.linesUpTo(m.cursor) > avail {
		m.scrollOffset++
		if m.scrollOffset > m.cursor {
			m.scrollOffset = m.cursor
			break
		}
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

// wrapText splits text into lines of at most width display columns.
// Word-wraps at spaces when possible; hard-breaks if a single word exceeds width.
// Returns a single-element slice with the original text if it fits within width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if lipgloss.Width(text) <= width {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var current string

	for _, word := range words {
		wordW := lipgloss.Width(word)

		// Word itself exceeds width — hard-break it.
		if wordW > width {
			// Flush current line if non-empty.
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			// Break the word into chunks of width.
			runes := []rune(word)
			for len(runes) > 0 {
				chunk := ""
				for len(runes) > 0 {
					candidate := chunk + string(runes[0])
					if lipgloss.Width(candidate) > width {
						break
					}
					chunk = candidate
					runes = runes[1:]
				}
				if chunk == "" && len(runes) > 0 {
					// Single rune wider than width — take it anyway to avoid infinite loop.
					chunk = string(runes[0])
					runes = runes[1:]
				}
				lines = append(lines, chunk)
			}
			continue
		}

		if current == "" {
			current = word
		} else if lipgloss.Width(current+" "+word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// Init satisfies the tea.Model interface for standalone use.
func (m todosModel) Init() tea.Cmd {
	return textinput.Blink
}

// Focused returns whether the pane is focused (used by lipgloss rendering).
func (m todosModel) Focused() bool {
	return m.focused
}

// StatusMsg returns the current transient status message (empty if none).
func (m todosModel) StatusMsg() string {
	return m.statusMsg
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
