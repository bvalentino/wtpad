package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestImporterEscEmitsExit(t *testing.T) {
	m := newImporterModel()
	m = m.open(80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a command")
	}
	if _, ok := cmd().(exitImportMsg); !ok {
		t.Fatalf("expected exitImportMsg, got %T", cmd())
	}
}

func TestImporterEnterEmptyPath(t *testing.T) {
	m := newImporterModel()
	m = m.open(80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with empty path should produce no command")
	}
}

func TestImporterRejectsNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	m := newImporterModel()
	m = m.open(80, 24)
	m.input.SetValue(path)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("non-.md file should not produce a command")
	}
	if m.errMsg == "" {
		t.Error("expected errMsg for non-.md file")
	}
}

func TestImporterTrimsTrailingDot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	os.WriteFile(path, []byte("content"), 0o644)

	m := newImporterModel()
	m = m.open(80, 24)
	m.input.SetValue(path + ".")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("trailing dot should be stripped, expected command")
	}
	msg, ok := cmd().(importFileMsg)
	if !ok {
		t.Fatalf("expected importFileMsg, got %T", cmd())
	}
	if msg.body != "content" {
		t.Errorf("body = %q, want %q", msg.body, "content")
	}
}

func TestImporterEnterValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	os.WriteFile(path, []byte("# Hello\nworld"), 0o644)

	m := newImporterModel()
	m = m.open(80, 24)
	m.input.SetValue(path)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter with valid file")
	}
	msg, ok := cmd().(importFileMsg)
	if !ok {
		t.Fatalf("expected importFileMsg, got %T", cmd())
	}
	if msg.body != "# Hello\nworld" {
		t.Errorf("body = %q, want %q", msg.body, "# Hello\nworld")
	}
}

func TestImporterEnterInvalidFile(t *testing.T) {
	m := newImporterModel()
	m = m.open(80, 24)
	m.input.SetValue("/nonexistent/path/to/file.md")

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("invalid file should not produce a command")
	}
	if m.errMsg == "" {
		t.Error("expected errMsg to be set for invalid file")
	}
}

func TestImporterTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// Create a temp file in a subdir of home
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	os.WriteFile(path, []byte("tilde test"), 0o644)

	// Only test expandTilde function directly
	expanded := expandTilde("~/something")
	if expanded != filepath.Join(home, "something") {
		t.Errorf("expandTilde(~/something) = %q, want %q", expanded, filepath.Join(home, "something"))
	}

	// Non-tilde path unchanged
	if got := expandTilde("/absolute/path"); got != "/absolute/path" {
		t.Errorf("expandTilde(/absolute/path) = %q", got)
	}
}

func TestImporterViewZeroDimensions(t *testing.T) {
	m := newImporterModel()
	if v := m.View(); v != "" {
		t.Errorf("expected empty view with zero dimensions, got %q", v)
	}
}

func TestImporterViewShowsError(t *testing.T) {
	m := newImporterModel()
	m = m.open(80, 24)
	m.errMsg = "Error: file not found"

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestImporterFooterHint(t *testing.T) {
	m := newImporterModel()
	if hint := m.FooterHint(); hint != "enter import · esc cancel" {
		t.Errorf("hint = %q", hint)
	}
}

func TestImporterOpenResetsState(t *testing.T) {
	m := newImporterModel()
	m.errMsg = "old error"
	m.input.SetValue("old path")

	m = m.open(80, 24)

	if m.errMsg != "" {
		t.Errorf("errMsg should be cleared, got %q", m.errMsg)
	}
	if m.input.Value() != "" {
		t.Errorf("input should be cleared, got %q", m.input.Value())
	}
	if m.width != 80 || m.height != 24 {
		t.Errorf("dimensions = %dx%d, want 80x24", m.width, m.height)
	}
}

func TestImporterResize(t *testing.T) {
	m := newImporterModel()
	m = m.open(80, 24)
	m = m.resize(120, 50)

	if m.width != 120 || m.height != 50 {
		t.Errorf("dimensions = %dx%d, want 120x50", m.width, m.height)
	}
}
