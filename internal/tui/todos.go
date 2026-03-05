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

// todoPrefixWidth is the display width of the checkbox prefix ("○ " / "✓ " / "▸ ").
var todoPrefixWidth = lipgloss.Width("○ ")

// TodoCounts holds the number of todos in each status.
type TodoCounts struct {
	Open, InProgress, Done int
}

// confirmKind represents which destructive action is pending confirmation.
type confirmKind int

const (
	confirmNone   confirmKind = iota
	confirmDelete
	confirmPurge
)

type todosModel struct {
	todos         []model.Todo
	store         *store.Store
	cursor        int
	scrollOffset  int
	width         int
	height        int
	textWidth     int // available columns for todo text (width minus prefix)
	focused       bool
	showCompleted bool // when true, show only done items; when false, show only open/in-progress
	inputActive   bool
	input         textinput.Model
	editIndex     int // -1 = adding new, >= 0 = editing existing
	statusMsg     string
	confirm       confirmKind
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

	// Handle delete/purge confirmation mode
	if m.confirm != confirmNone {
		switch keyMsg.String() {
		case "y":
			switch m.confirm {
			case confirmDelete:
				m = m.deleteCurrent()
			case confirmPurge:
				if m.showCompleted {
					m = m.purgeDone()
					m.showCompleted = false
					m = m.snapCursor()
					m = m.adjustScroll()
				} else {
					m = m.purgeOpen()
				}
			}
		default:
			// any other key cancels
		}
		m.confirm = confirmNone
		return m, nil
	}

	switch keyMsg.String() {
	case "down":
		m = m.moveCursor(1)
	case "up":
		m = m.moveCursor(-1)
	case "a":
		if m.showCompleted {
			return m, nil // no adding in completed view
		}
		m.input.SetValue("")
		m.input.Focus()
		m.inputActive = true
		m.editIndex = -1
		return m, func() tea.Msg { return enterInputMsg{} }
	case "enter":
		if len(m.todos) > 0 {
			m.input.SetValue(m.todos[m.cursor].Text)
			m.input.CursorEnd() // SetValue preserves stale cursor position; force to end
			m.input.Focus()
			m.inputActive = true
			m.editIndex = m.cursor
			return m, func() tea.Msg { return enterInputMsg{} }
		}
	case " ":
		m = m.toggleDone()
	case "i":
		m = m.toggleInProgress()
	case "x", "delete":
		if len(m.todos) > 0 {
			m.confirm = confirmDelete
		}
	case "J":
		m = m.moveTodo(1)
	case "K":
		m = m.moveTodo(-1)
	case "X":
		m.confirm = confirmPurge
	case "v":
		m.showCompleted = !m.showCompleted
		m.cursor = 0
		m.scrollOffset = 0
		m = m.snapCursor()
		m = m.adjustScroll()
	case "c":
		if len(m.todos) > 0 {
			if err := clipboard.WriteAll(m.todos[m.cursor].Text); err != nil {
				m.statusMsg = "Copy failed"
			} else {
				m.statusMsg = "Copied!"
			}
			return m, clearStatusAfter(2 * time.Second)
		}
	case "T":
		return m, func() tea.Msg { return enterTemplateMsg{saving: false} }
	case "S":
		return m, func() tea.Msg { return enterTemplateMsg{saving: true} }
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
					// Move cursor to the newly added todo.
					// Scan in reverse: sortTodos is stable and the new
					// item was appended last, so it's the last match.
					for i := len(m.todos) - 1; i >= 0; i-- {
						if m.todos[i].Text == text && m.todos[i].Status == model.StatusOpen {
							m.cursor = i
							break
						}
					}
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
	// Check if there are any visible items.
	hasVisible := false
	for _, t := range m.todos {
		if m.isVisible(t) {
			hasVisible = true
			break
		}
	}

	if !hasVisible && !m.inputActive {
		if m.showCompleted {
			return renderEmptyState(m.width, m.height, []string{
				hintStyle.Render("No completed todos"),
			})
		}
		if len(m.todos) == 0 {
			return renderEmptyState(m.width, m.height, []string{
				"Track tasks",
				hintStyle.Render("Press 'a' to add your first one."),
			})
		}
		return renderEmptyState(m.width, m.height, []string{
			hintStyle.Render("No open todos"),
		})
	}

	var b strings.Builder
	linesUsed := 0
	visibleLines := m.height - 2 // reserve 2 lines for bottom bar (divider + hint/input/confirm)
	if visibleLines < 1 {
		visibleLines = 1
	}

	prevNotDone := false
	indent := strings.Repeat(" ", todoPrefixWidth)

	for i := m.scrollOffset; i < len(m.todos) && linesUsed < visibleLines; i++ {
		todo := m.todos[i]

		// Skip items not matching the current view.
		if !m.isVisible(todo) {
			continue
		}

		// Blank line between non-done items for breathing room.
		if todo.Status != model.StatusDone && prevNotDone && linesUsed < visibleLines {
			if linesUsed > 0 {
				b.WriteString("\n")
				linesUsed++
			}
		}

		if linesUsed >= visibleLines {
			break
		}

		// Newline before the item (except first rendered line).
		// This \n is a line terminator, not a visible line — don't increment linesUsed.
		if linesUsed > 0 {
			b.WriteString("\n")
		}

		// Render the todo line(s) with wrapping.
		var prefix string
		switch todo.Status {
		case model.StatusDone:
			prefix = "✓ "
		case model.StatusInProgress:
			prefix = "▸ "
		default:
			prefix = "○ "
		}

		wrapped := wrapText(todo.Text, m.textWidth)
		selected := i == m.cursor && m.focused

		// Pick base style from status, then compose selection background.
		var style lipgloss.Style
		styled := false
		switch todo.Status {
		case model.StatusDone:
			style = todoDone
			styled = true
		case model.StatusInProgress:
			style = todoInProgress
			styled = true
		}
		if selected {
			style = style.Reverse(true)
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
		prevNotDone = todo.Status != model.StatusDone
	}

	// Build bottom bar content.
	var bar strings.Builder
	bar.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)))
	bar.WriteString("\n")
	switch {
	case m.inputActive:
		bar.WriteString(m.input.View())
	case m.confirm == confirmDelete:
		bar.WriteString(listConfirm.Render("Delete todo? (y to confirm)"))
	case m.confirm == confirmPurge && m.showCompleted:
		bar.WriteString(listConfirm.Render("Clear all completed? (y to confirm)"))
	case m.confirm == confirmPurge && !m.showCompleted:
		bar.WriteString(listConfirm.Render("Clear all open todos? (y to confirm)"))
	case !m.showCompleted:
		bar.WriteString(hintStyle.Render("Add Todo (a)"))
	}

	// Assemble: item lines, then pad to fill visibleLines, then bottom bar.
	// Total output must be exactly m.height lines (visibleLines + 2).
	itemContent := b.String()
	itemLines := strings.Split(itemContent, "\n")
	// Pad item lines to exactly visibleLines entries.
	for len(itemLines) < visibleLines {
		itemLines = append(itemLines, "")
	}
	itemLines = itemLines[:visibleLines] // trim if over (shouldn't happen)
	// Append the 2 bottom bar lines.
	barLines := strings.Split(bar.String(), "\n")
	itemLines = append(itemLines, barLines...)

	return strings.Join(itemLines, "\n")
}

// moveCursor moves the cursor to the next visible item in the given direction.
// If no visible item exists in that direction, returns unchanged (no-op).
func (m todosModel) moveCursor(delta int) todosModel {
	if len(m.todos) == 0 {
		return m
	}
	// Step in the given direction until we find a visible item.
	newIdx := m.cursor
	for range len(m.todos) {
		newIdx += delta
		if newIdx < 0 || newIdx >= len(m.todos) {
			return m // hit bounds, stay put
		}
		if m.isVisible(m.todos[newIdx]) {
			m.cursor = newIdx
			m = m.adjustScroll()
			return m
		}
	}
	return m
}

// moveTodo swaps the current todo with its neighbor in the given direction
// (+1 = down, -1 = up) within the same status group. No-op at group boundaries.
func (m todosModel) moveTodo(delta int) todosModel {
	if len(m.todos) == 0 {
		return m
	}
	if m.showCompleted {
		return m // reordering not supported in completed view
	}
	newIdx := m.cursor + delta
	if newIdx < 0 || newIdx >= len(m.todos) {
		return m
	}
	if m.todos[newIdx].Status != m.todos[m.cursor].Status {
		return m
	}
	m.todos[m.cursor], m.todos[newIdx] = m.todos[newIdx], m.todos[m.cursor]
	m.cursor = newIdx
	m = m.adjustScroll()
	m.save()
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

// snapCursor clamps cursor to valid bounds, then moves it to the nearest
// visible item if the current position is hidden.
func (m todosModel) snapCursor() todosModel {
	m = m.clampCursor()
	return m.clampCursorVisible()
}

// clampCursorVisible moves cursor to the nearest visible item.
// If the current cursor points to a hidden item, scan forward then backward.
func (m todosModel) clampCursorVisible() todosModel {
	if len(m.todos) == 0 {
		m.cursor = 0
		return m
	}
	// If current position is visible, we're fine.
	if m.cursor < len(m.todos) && m.isVisible(m.todos[m.cursor]) {
		return m
	}
	// Scan forward for a visible item.
	for i := m.cursor; i < len(m.todos); i++ {
		if m.isVisible(m.todos[i]) {
			m.cursor = i
			return m
		}
	}
	// Scan backward.
	for i := m.cursor - 1; i >= 0; i-- {
		if m.isVisible(m.todos[i]) {
			m.cursor = i
			return m
		}
	}
	// No visible items — cursor stays at 0.
	m.cursor = 0
	return m
}

// availableLines returns the number of lines available for todo rendering.
// Always reserves 2 lines for the bottom bar (divider + hint/input/confirm).
func (m todosModel) availableLines() int {
	h := m.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

// linesUpTo counts rendered lines from scrollOffset through targetIdx,
// mirroring the View() line-accounting logic (blank lines between items,
// wrapped item heights). Does not account for the bottom action bar.
// Must stay in sync with View() — any change to spacing or wrapping
// in View() must be reflected here.
func (m todosModel) linesUpTo(targetIdx int) int {
	linesUsed := 0
	prevNotDone := false
	avail := m.availableLines()

	for i := m.scrollOffset; i < len(m.todos) && i <= targetIdx; i++ {
		todo := m.todos[i]

		// Skip items not matching the current view.
		if !m.isVisible(todo) {
			continue
		}

		// Blank line between consecutive non-done items.
		if todo.Status != model.StatusDone && prevNotDone && linesUsed < avail {
			if linesUsed > 0 {
				linesUsed++
			}
		}

		// Newline before item is a terminator (no count).

		// The item itself — may occupy multiple lines when wrapped.
		linesUsed += len(wrapText(todo.Text, m.textWidth))
		prevNotDone = todo.Status != model.StatusDone
	}

	return linesUsed
}

// adjustScroll ensures scrollOffset keeps the cursor visible within the pane.
// Uses line counting to account for blank lines and wrapped item heights.
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

	// Scroll back up to fill viewport when there's trailing whitespace,
	// but never push the cursor out of view.
	for m.scrollOffset > 0 && m.linesUpTo(len(m.todos)-1) < avail {
		m.scrollOffset--
		if m.linesUpTo(m.cursor) > avail {
			m.scrollOffset++
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
	t := &m.todos[m.cursor]
	if t.Status == model.StatusDone {
		t.Status = model.StatusOpen
	} else {
		t.Status = model.StatusDone
	}
	m.todos = sortTodos(m.todos)
	m = m.snapCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// toggleInProgress toggles InProgress on the selected todo, re-sorts, and saves.
// Only toggles open ↔ in-progress. Done items are not affected (use d/Space first).
func (m todosModel) toggleInProgress() todosModel {
	if len(m.todos) == 0 {
		return m
	}
	t := &m.todos[m.cursor]
	if t.Status == model.StatusDone {
		return m
	}
	if t.Status == model.StatusInProgress {
		t.Status = model.StatusOpen
	} else {
		t.Status = model.StatusInProgress
	}
	m.todos = sortTodos(m.todos)
	m = m.snapCursor()
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
	m = m.snapCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// purgeDone removes all completed todos and saves.
func (m todosModel) purgeDone() todosModel {
	filtered := make([]model.Todo, 0, len(m.todos))
	for _, t := range m.todos {
		if t.Status != model.StatusDone {
			filtered = append(filtered, t)
		}
	}
	m.todos = filtered
	m = m.snapCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// purgeOpen removes all non-done todos and saves.
func (m todosModel) purgeOpen() todosModel {
	filtered := make([]model.Todo, 0, len(m.todos))
	for _, t := range m.todos {
		if t.Status == model.StatusDone {
			filtered = append(filtered, t)
		}
	}
	m.todos = filtered
	m = m.snapCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// ImportTodos appends the given todos to the current list, re-sorts, and saves.
func (m todosModel) ImportTodos(todos []model.Todo) todosModel {
	m.todos = append(m.todos, todos...)
	m.todos = sortTodos(m.todos)
	m = m.snapCursor()
	m = m.adjustScroll()
	m.save()
	return m
}

// OpenTodos returns all non-done todos (open + in-progress).
func (m todosModel) OpenTodos() []model.Todo {
	var result []model.Todo
	for _, t := range m.todos {
		if t.Status != model.StatusDone {
			result = append(result, t)
		}
	}
	return result
}

// save persists todos to disk, logging on failure.
func (m todosModel) save() {
	if err := m.store.SaveTodos(m.todos); err != nil {
		log.Printf("wtpad: failed to save todos: %v", err)
	}
}

// sortTodos returns todos grouped: in-progress first, then open, then done,
// preserving relative order within each group.
func sortTodos(todos []model.Todo) []model.Todo {
	inProgress := make([]model.Todo, 0)
	open := make([]model.Todo, 0, len(todos))
	done := make([]model.Todo, 0)
	for _, t := range todos {
		switch t.Status {
		case model.StatusDone:
			done = append(done, t)
		case model.StatusInProgress:
			inProgress = append(inProgress, t)
		default:
			open = append(open, t)
		}
	}
	result := make([]model.Todo, 0, len(todos))
	result = append(result, inProgress...)
	result = append(result, open...)
	result = append(result, done...)
	return result
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

// isVisible returns whether a todo should be shown in the current view mode.
func (m todosModel) isVisible(t model.Todo) bool {
	if m.showCompleted {
		return t.Status == model.StatusDone
	}
	return t.Status != model.StatusDone
}

// ShowingCompleted returns whether the completed view is active.
func (m todosModel) ShowingCompleted() bool {
	return m.showCompleted
}

// FooterHint returns the mode-aware hint string for the footer.
func (m todosModel) FooterHint() string {
	if m.showCompleted {
		return "v back · ? help · tab switch"
	}
	return "? help · tab switch"
}

// Counts returns the number of todos in each status.
func (m todosModel) Counts() TodoCounts {
	var c TodoCounts
	for _, t := range m.todos {
		switch t.Status {
		case model.StatusDone:
			c.Done++
		case model.StatusInProgress:
			c.InProgress++
		default:
			c.Open++
		}
	}
	return c
}
