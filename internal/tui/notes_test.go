package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func tempNotesStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

func seedNotes(t *testing.T, s *store.Store, names []string, bodies []string) []model.Note {
	t.Helper()
	for i, name := range names {
		if _, err := s.SaveNote(name, bodies[i]); err != nil {
			t.Fatalf("SaveNote(%s): %v", name, err)
		}
	}
	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	return notes
}

func TestNotesEmptyView(t *testing.T) {
	m := newNotes(nil, nil)
	m = m.SetSize(40, 10)
	view := m.View()
	if !strings.Contains(view, "No notes") {
		t.Errorf("empty view should show placeholder, got %q", view)
	}
}

func TestNotesHeaderTimestamp(t *testing.T) {
	ts, _ := time.Parse("20060102-150405", "20260228-143000")
	note := model.Note{Name: "20260228-143000", CreatedAt: ts, Body: "some body"}
	m := newNotes(nil, nil)
	header := m.noteHeaderText(note)
	if header != "Feb 28 14:30" {
		t.Errorf("header = %q, want %q", header, "Feb 28 14:30")
	}
}

func TestNotesHeaderFromMarkdownTitle(t *testing.T) {
	note := model.Note{Name: "20260228-143000", Body: "# My Title\nsome content"}
	m := newNotes(nil, nil)
	header := m.noteHeaderText(note)
	if header != "My Title" {
		t.Errorf("header = %q, want %q", header, "My Title")
	}
}

func TestNotesNavigation(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260101-100000", "20260201-100000", "20260301-100000"},
		[]string{"note 1", "note 2", "note 3"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("after j cursor = %d, want 1", m.cursor)
	}

	// Move down again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("after j cursor = %d, want 2", m.cursor)
	}

	// Can't go past end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("should clamp at end, cursor = %d, want 2", m.cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("after k cursor = %d, want 1", m.cursor)
	}
}

func TestNotesNewNoteSignal(t *testing.T) {
	m := newNotes(nil, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("'a' should produce a command")
	}
	msg := cmd()
	editorMsg, ok := msg.(enterEditorMsg)
	if !ok {
		t.Fatalf("expected enterEditorMsg, got %T", msg)
	}
	if editorMsg.name != "" {
		t.Errorf("new note should have empty name, got %q", editorMsg.name)
	}
}

func TestNotesEditSignal(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260228-100000"},
		[]string{"hello world"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("'e' should produce a command")
	}
	msg := cmd()
	editorMsg, ok := msg.(enterEditorMsg)
	if !ok {
		t.Fatalf("expected enterEditorMsg, got %T", msg)
	}
	if editorMsg.name != "20260228-100000" {
		t.Errorf("edit msg name = %q, want %q", editorMsg.name, "20260228-100000")
	}
	if editorMsg.body != "hello world" {
		t.Errorf("edit msg body = %q, want %q", editorMsg.body, "hello world")
	}
}

func TestNotesDeleteConfirmation(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260228-100000"},
		[]string{"to delete"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press x — should show confirmation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !m.confirmDelete {
		t.Fatal("expected confirmDelete to be true after 'x'")
	}

	// View should show confirmation text
	view := m.View()
	if !strings.Contains(view, "Delete note?") {
		t.Errorf("view should show delete confirmation, got %q", view)
	}

	// Press 'n' to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.confirmDelete {
		t.Error("confirmation should be cancelled")
	}
	if len(m.notes) != 1 {
		t.Errorf("note should not be deleted, got %d notes", len(m.notes))
	}
}

func TestNotesDeleteConfirm(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260228-100000"},
		[]string{"to delete"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press x then y to confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.confirmDelete {
		t.Error("confirmDelete should be false after confirmation")
	}
	if len(m.notes) != 0 {
		t.Errorf("note should be deleted, got %d notes", len(m.notes))
	}

	// Verify it's gone from disk
	diskNotes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(diskNotes) != 0 {
		t.Errorf("note should be deleted from disk, got %d notes", len(diskNotes))
	}
}

func TestNotesRenderPreview(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260228-100000"},
		[]string{"# My Note\nline one\nline two\nline three"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	view := m.View()
	if !strings.Contains(view, "My Note") {
		t.Errorf("view should show note title, got %q", view)
	}
}

func TestNotesScrollWithFixedHeight(t *testing.T) {
	s := tempNotesStore(t)
	// Each note takes 2 lines (header + 1 body line)
	notes := seedNotes(t, s,
		[]string{"20260101-100000", "20260201-100000", "20260301-100000", "20260401-100000"},
		[]string{"line1\nline2\nline3", "line1\nline2\nline3", "line1\nline2\nline3", "line1\nline2\nline3"},
	)

	m := newNotes(notes, s)
	// Height=4 fits 2 notes (2 lines each)
	m = m.SetSize(40, 4)
	m = m.SetFocus(true)

	// Navigate to 3rd note — should scroll since first 2 fill the viewport
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced, still 0")
	}
}

func TestNotesSelectedNotExpanded(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260228-100000"},
		[]string{"# My Note\nfirst line\nsecond line\nthird line"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	view := m.View()
	if strings.Contains(view, "second line") {
		t.Errorf("selected note should not show second line, got %q", view)
	}
	if strings.Contains(view, "third line") {
		t.Errorf("selected note should not show third line, got %q", view)
	}
	if !strings.Contains(view, "first line") {
		t.Errorf("selected note should show first line, got %q", view)
	}
}

func TestNotesSetNotes(t *testing.T) {
	m := newNotes(nil, nil)
	m = m.SetSize(40, 10)
	m.cursor = 5 // out of range

	notes := []model.Note{
		{Name: "20260301-100000"},
		{Name: "20260201-100000"},
	}
	m = m.SetNotes(notes)

	if m.cursor != 1 {
		t.Errorf("cursor should be clamped to %d, got %d", 1, m.cursor)
	}
	if len(m.notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(m.notes))
	}
}
