package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestViewerZeroDimensions(t *testing.T) {
	var v viewerModel
	v = v.openViewer("20260228-143000", "# Hello\nworld", 0, 0)
	out := v.View()
	if out != "" {
		t.Errorf("expected empty string for zero dimensions, got %q", out)
	}
}

func TestViewerShowsBody(t *testing.T) {
	var v viewerModel
	v = v.openViewer("20260228-143000", "# Hello\nsome **bold** text", 60, 20)
	out := v.View()
	if !strings.Contains(out, "Hello") {
		t.Errorf("output should contain heading text, got %q", out)
	}
	if !strings.Contains(out, "bold") {
		t.Errorf("output should contain body text, got %q", out)
	}
}

func TestViewerHeaderTimestamp(t *testing.T) {
	var v viewerModel
	v = v.openViewer("20260228-143000", "hello", 60, 20)
	out := v.View()
	if !strings.Contains(out, "Feb 28 14:30") {
		t.Errorf("view should show formatted timestamp in top border, got %q", out)
	}
	if !strings.Contains(out, "╭") {
		t.Error("view should show top border")
	}
}

func TestViewerEscExits(t *testing.T) {
	var v viewerModel
	v = v.openViewer("test", "body", 60, 20)

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(exitViewerMsg); !ok {
		t.Fatalf("expected exitViewerMsg, got %T", msg)
	}
}

func TestViewerQExits(t *testing.T) {
	var v viewerModel
	v = v.openViewer("test", "body", 60, 20)

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(exitViewerMsg); !ok {
		t.Fatalf("expected exitViewerMsg, got %T", msg)
	}
}

func TestViewerEOpensEditor(t *testing.T) {
	var v viewerModel
	v = v.openViewer("20260228-143000", "hello world", 60, 20)

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("e should produce a command")
	}
	msg := cmd()
	editorMsg, ok := msg.(enterEditorMsg)
	if !ok {
		t.Fatalf("expected enterEditorMsg, got %T", msg)
	}
	if editorMsg.name != "20260228-143000" {
		t.Errorf("editor msg name = %q, want %q", editorMsg.name, "20260228-143000")
	}
	if editorMsg.body != "hello world" {
		t.Errorf("editor msg body = %q, want %q", editorMsg.body, "hello world")
	}
}

func TestViewerScrollDown(t *testing.T) {
	body := strings.Repeat("line\n", 100)

	var v viewerModel
	v = v.openViewer("test", body, 60, 5)

	if v.scrollOffset != 0 {
		t.Fatalf("initial scroll = %d, want 0", v.scrollOffset)
	}

	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.scrollOffset != 1 {
		t.Errorf("after down scroll = %d, want 1", v.scrollOffset)
	}
}

func TestViewerScrollUpClamps(t *testing.T) {
	var v viewerModel
	v = v.openViewer("test", "short", 60, 20)

	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyUp})
	if v.scrollOffset != 0 {
		t.Errorf("scroll should not go negative, got %d", v.scrollOffset)
	}
}

func TestViewerPageScroll(t *testing.T) {
	body := strings.Repeat("line\n", 200)

	var v viewerModel
	v = v.openViewer("test", body, 60, 10)

	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if v.scrollOffset == 0 {
		t.Error("pgdown should advance scroll offset")
	}
	contentH := overlayContentHeight(v.height)
	if v.scrollOffset != contentH {
		t.Errorf("pgdown scroll = %d, want %d", v.scrollOffset, contentH)
	}
}

func TestViewerScrollClampsToMax(t *testing.T) {
	var v viewerModel
	v = v.openViewer("test", "one\ntwo\nthree", 60, 20)

	for i := 0; i < 50; i++ {
		v, _ = v.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if v.scrollOffset < 0 {
		t.Errorf("scroll should not be negative, got %d", v.scrollOffset)
	}
}

func TestViewerHintBar(t *testing.T) {
	var v viewerModel
	v = v.openViewer("test", "hello", 60, 20)
	out := v.View()
	if !strings.Contains(out, "Edit") {
		t.Errorf("view should show edit hint, got %q", out)
	}
	if !strings.Contains(out, "Back") {
		t.Errorf("view should show back hint, got %q", out)
	}
}

func TestViewerResize(t *testing.T) {
	var v viewerModel
	v = v.openViewer("test", "# Hello\nworld", 60, 20)

	v = v.resize(40, 15)
	if v.width != 40 || v.height != 15 {
		t.Errorf("dimensions = %d×%d, want 40×15", v.width, v.height)
	}
	if len(v.lines) == 0 {
		t.Error("lines should not be empty after resize")
	}
}

func TestNotesEnterOpensViewer(t *testing.T) {
	s := tempNotesStore(t)
	notes := seedNotes(t, s,
		[]string{"20260228-100000"},
		[]string{"hello world"},
	)

	m := newNotes(notes, s)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	viewerMsg, ok := msg.(enterViewerMsg)
	if !ok {
		t.Fatalf("expected enterViewerMsg, got %T", msg)
	}
	if viewerMsg.name != "20260228-100000" {
		t.Errorf("viewer msg name = %q, want %q", viewerMsg.name, "20260228-100000")
	}
	if viewerMsg.body != "hello world" {
		t.Errorf("viewer msg body = %q, want %q", viewerMsg.body, "hello world")
	}
}
