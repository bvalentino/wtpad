package tui

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// enterEditorMsg signals root to switch to modeEditor.
// Name is empty for a new note, non-empty for editing an existing note.
type enterEditorMsg struct {
	name string
	body string
}

type notesModel struct {
	notes          []model.Note
	store          *store.Store
	cursor         int
	scrollOffset   int
	width          int
	height         int
	focused        bool
	confirmDelete bool // true when showing delete confirmation prompt
}

func newNotes(notes []model.Note, s *store.Store) notesModel {
	m := notesModel{
		notes: notes,
		store: s,
	}
	m = m.loadAllBodies()
	return m
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
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// Handle delete confirmation mode
	if m.confirmDelete {
		switch keyMsg.String() {
		case "y":
			m = m.deleteSelected()
			m.confirmDelete = false
		default:
			m.confirmDelete = false
		}
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		m = m.moveCursor(1)
	case "k", "up":
		m = m.moveCursor(-1)
	case "n", "a":
		return m, func() tea.Msg { return enterEditorMsg{} }
	case "e", "enter":
		if len(m.notes) > 0 {
			note := m.notes[m.cursor]
			return m, func() tea.Msg {
				return enterEditorMsg{name: note.Name, body: note.Body}
			}
		}
	case "x", "delete":
		if len(m.notes) > 0 {
			m.confirmDelete = true
		}
	}

	return m, nil
}

func (m notesModel) View() string {
	if len(m.notes) == 0 {
		return "No notes yet. Press 'a' to create one."
	}

	var b strings.Builder

	// Reserve 1 line for confirmation prompt if active
	visibleLines := m.height
	if m.confirmDelete {
		visibleLines--
	}
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Render notes with fixed line heights
	linesUsed := 0
	for i := m.scrollOffset; i < len(m.notes) && linesUsed < visibleLines; i++ {
		note := m.notes[i]
		selected := i == m.cursor && m.focused

		header := m.noteHeaderText(note)
		lines := m.noteLines(note)

		// Header line
		headerLine := noteHeader.Render(header)
		if selected {
			headerLine = noteSelected.Render(headerLine)
		}
		if linesUsed > 0 {
			b.WriteString("\n")
		}
		b.WriteString(headerLine)
		linesUsed++

		// Body lines
		for _, line := range lines {
			if linesUsed >= visibleLines {
				break
			}
			b.WriteString("\n")
			rendered := notePreview.Render(line)
			if selected {
				rendered = noteSelected.Render(rendered)
			}
			b.WriteString(rendered)
			linesUsed++
		}
	}

	// Add note hint
	if linesUsed > 0 && linesUsed < visibleLines {
		b.WriteString("\n")
		linesUsed++
		if linesUsed < visibleLines {
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("Add note (a)"))
			linesUsed++
		}
		if linesUsed < visibleLines {
			b.WriteString("\n")
			linesUsed++
		}
	}

	if m.confirmDelete {
		b.WriteString("\n")
		b.WriteString(noteConfirm.Render("Delete note? (y to confirm)"))
	}

	return b.String()
}

// splitHeadingAndBody extracts a markdown heading from the first line of body.
// If the first line starts with "# ", it returns the heading text and remaining body.
// Otherwise heading is empty and body is returned unchanged.
func splitHeadingAndBody(body string) (heading, rest string, hasHeading bool) {
	if body == "" {
		return "", "", false
	}
	firstLine := strings.SplitN(body, "\n", 2)[0]
	if !strings.HasPrefix(firstLine, "# ") {
		return "", body, false
	}
	heading = firstLine[2:]
	if parts := strings.SplitN(body, "\n", 2); len(parts) > 1 {
		rest = strings.TrimLeft(parts[1], "\n")
	}
	return heading, rest, true
}

// noteHeaderText returns the display header for a note.
// Uses the first line if it starts with "# ", otherwise formats the timestamp.
func (m notesModel) noteHeaderText(note model.Note) string {
	if heading, _, has := splitHeadingAndBody(note.Body); has {
		return heading
	}
	if !note.CreatedAt.IsZero() {
		return note.CreatedAt.Format("Jan 02 15:04")
	}
	return note.Name
}

// noteLines returns the first line of the note body, truncated.
// Always shows exactly one line regardless of selection state.
func (m notesModel) noteLines(note model.Note) []string {
	_, body, _ := splitHeadingAndBody(note.Body)
	if body == "" {
		return nil
	}

	firstLine := strings.SplitN(body, "\n", 2)[0]
	line := truncate(firstLine, m.width)
	if strings.Contains(body, "\n") {
		line = truncate(firstLine, m.width-1) + "…"
	}
	return []string{line}
}

// moveCursor moves the cursor by delta, clamps, and adjusts scroll.
func (m notesModel) moveCursor(delta int) notesModel {
	m.cursor += delta
	m = m.clampCursor()
	m = m.adjustScroll()
	return m
}

// clampCursor ensures cursor is within [0, len(notes)-1].
func (m notesModel) clampCursor() notesModel {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if max := len(m.notes) - 1; m.cursor > max {
		if max < 0 {
			m.cursor = 0
		} else {
			m.cursor = max
		}
	}
	return m
}

// noteHeight returns the number of terminal lines a note occupies.
// 1 for header only (no body), 2 for header + 1 body line.
func (m notesModel) noteHeight(idx int) int {
	note := m.notes[idx]
	_, body, _ := splitHeadingAndBody(note.Body)
	if body == "" {
		return 1
	}
	return 2
}

// availableLines returns the number of lines available for note rendering.
func (m notesModel) availableLines() int {
	h := m.height
	if m.confirmDelete {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// adjustScroll ensures scrollOffset keeps the cursor visible,
// accounting for variable-height note entries and the confirmation prompt.
func (m notesModel) adjustScroll() notesModel {
	if m.height < 1 || len(m.notes) == 0 {
		return m
	}
	// Scroll up if cursor is above viewport
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	// Scroll down if cursor is below viewport — sum heights from scrollOffset
	avail := m.availableLines()
	for {
		used := 0
		cursorVisible := false
		for i := m.scrollOffset; i < len(m.notes); i++ {
			h := m.noteHeight(i)
			if used+h > avail && i > m.scrollOffset {
				break
			}
			used += h
			if i == m.cursor {
				cursorVisible = true
				break
			}
		}
		if cursorVisible {
			break
		}
		m.scrollOffset++
		if m.scrollOffset > m.cursor {
			m.scrollOffset = m.cursor
			break
		}
	}
	return m
}

// loadAllBodies loads the body of every note from disk.
func (m notesModel) loadAllBodies() notesModel {
	if m.store == nil {
		return m
	}
	for i := range m.notes {
		if m.notes[i].Body != "" {
			continue
		}
		loaded, err := m.store.LoadNote(m.notes[i].Name)
		if err != nil {
			log.Printf("wtpad: failed to load note %s: %v", m.notes[i].Name, err)
			continue
		}
		m.notes[i].Body = loaded.Body
	}
	return m
}

// deleteSelected removes the selected note from disk and the slice.
func (m notesModel) deleteSelected() notesModel {
	if len(m.notes) == 0 {
		return m
	}
	name := m.notes[m.cursor].Name
	if err := m.store.DeleteNote(name); err != nil {
		log.Printf("wtpad: failed to delete note %s: %v", name, err)
		return m
	}
	m.notes = append(m.notes[:m.cursor], m.notes[m.cursor+1:]...)
	m = m.clampCursor()
	m = m.adjustScroll()
	return m
}

// SetNotes replaces the notes slice (used after editor saves a new/updated note).
func (m notesModel) SetNotes(notes []model.Note) notesModel {
	m.notes = notes
	m = m.loadAllBodies()
	m = m.clampCursor()
	m = m.adjustScroll()
	return m
}

// Focused returns whether the pane is focused.
func (m notesModel) Focused() bool {
	return m.focused
}

// Init satisfies the tea.Model interface for standalone use.
func (m notesModel) Init() tea.Cmd {
	return nil
}

// SelectedNote returns the currently selected note, or nil if empty.
func (m notesModel) SelectedNote() *model.Note {
	if len(m.notes) == 0 {
		return nil
	}
	return &m.notes[m.cursor]
}

