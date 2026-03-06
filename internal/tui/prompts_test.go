package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func tempPromptStoreForTUI(t *testing.T) *store.PromptStore {
	t.Helper()
	return store.NewPromptStore(filepath.Join(t.TempDir(), "prompts"))
}

func seedPrompts(t *testing.T, ps *store.PromptStore, names []string, bodies []string) []model.Note {
	t.Helper()
	for i, name := range names {
		if _, err := ps.SavePrompt(name, bodies[i]); err != nil {
			t.Fatalf("SavePrompt(%s): %v", name, err)
		}
	}
	prompts, err := ps.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	return prompts
}

func TestPromptsEmptyView(t *testing.T) {
	m := newPrompts(nil, nil)
	m = m.SetSize(40, 10)
	view := m.View()
	if !strings.Contains(view, "Reusable text snippets") {
		t.Errorf("empty view should describe prompts, got %q", view)
	}
	if !strings.Contains(view, "Press 'a'") {
		t.Errorf("empty view should show create hint, got %q", view)
	}
}

func TestPromptsNavigation(t *testing.T) {
	ps := tempPromptStoreForTUI(t)
	prompts := seedPrompts(t, ps,
		[]string{"20260101-100000", "20260201-100000", "20260301-100000"},
		[]string{"prompt 1", "prompt 2", "prompt 3"},
	)

	m := newPrompts(prompts, ps)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("after down cursor = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("after down cursor = %d, want 2", m.cursor)
	}

	// Can't go past end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("should clamp at end, cursor = %d, want 2", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("after up cursor = %d, want 1", m.cursor)
	}
}

func TestPromptsNewPromptSignal(t *testing.T) {
	m := newPrompts(nil, nil)
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
		t.Errorf("new prompt should have empty name, got %q", editorMsg.name)
	}
}

func TestPromptsEditSignal(t *testing.T) {
	ps := tempPromptStoreForTUI(t)
	prompts := seedPrompts(t, ps,
		[]string{"20260228-100000"},
		[]string{"hello world"},
	)

	m := newPrompts(prompts, ps)
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

func TestPromptsViewerSignal(t *testing.T) {
	ps := tempPromptStoreForTUI(t)
	prompts := seedPrompts(t, ps,
		[]string{"20260228-100000"},
		[]string{"view me"},
	)

	m := newPrompts(prompts, ps)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("'enter' should produce a command")
	}
	msg := cmd()
	viewerMsg, ok := msg.(enterViewerMsg)
	if !ok {
		t.Fatalf("expected enterViewerMsg, got %T", msg)
	}
	if viewerMsg.name != "20260228-100000" {
		t.Errorf("viewer msg name = %q, want %q", viewerMsg.name, "20260228-100000")
	}
	if viewerMsg.body != "view me" {
		t.Errorf("viewer msg body = %q, want %q", viewerMsg.body, "view me")
	}
}

func TestPromptsCopyToClipboard(t *testing.T) {
	if clipboard.Unsupported {
		t.Skip("clipboard not available in this environment")
	}

	ps := tempPromptStoreForTUI(t)
	prompts := seedPrompts(t, ps,
		[]string{"20260228-100000"},
		[]string{"copy this text"},
	)

	m := newPrompts(prompts, ps)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press 'c' — returns an async clipboard write command
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if cmd == nil {
		t.Fatal("'c' should produce a command")
	}
	if m.statusMsg != "Copying…" {
		t.Errorf("statusMsg = %q, want %q", m.statusMsg, "Copying…")
	}

	// Execute the command to perform the clipboard write
	resultMsg := cmd()

	// Feed the result back to Update
	m, _ = m.Update(resultMsg)
	if m.statusMsg != "Copied!" {
		t.Errorf("statusMsg = %q, want %q", m.statusMsg, "Copied!")
	}

	got, err := clipboard.ReadAll()
	if err != nil {
		t.Fatalf("clipboard.ReadAll: %v", err)
	}
	if got != "copy this text" {
		t.Errorf("clipboard = %q, want %q", got, "copy this text")
	}
}

func TestPromptsDeleteConfirmation(t *testing.T) {
	ps := tempPromptStoreForTUI(t)
	prompts := seedPrompts(t, ps,
		[]string{"20260228-100000"},
		[]string{"to delete"},
	)

	m := newPrompts(prompts, ps)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if m.confirm != confirmDelete {
		t.Fatal("expected confirm == confirmDelete after delete key")
	}

	view := m.View()
	if !strings.Contains(view, "Delete prompt?") {
		t.Errorf("view should show delete confirmation, got %q", view)
	}

	// Cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.confirm != confirmNone {
		t.Error("confirmation should be cancelled")
	}
	if len(m.items) != 1 {
		t.Errorf("prompt should not be deleted, got %d prompts", len(m.items))
	}
}

func TestPromptsDeleteConfirm(t *testing.T) {
	ps := tempPromptStoreForTUI(t)
	prompts := seedPrompts(t, ps,
		[]string{"20260228-100000"},
		[]string{"to delete"},
	)

	m := newPrompts(prompts, ps)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.confirm != confirmNone {
		t.Error("confirm should be confirmNone after confirmation")
	}
	if len(m.items) != 0 {
		t.Errorf("prompt should be deleted, got %d prompts", len(m.items))
	}
}

func TestPromptsSetPrompts(t *testing.T) {
	m := newPrompts(nil, nil)
	m = m.SetSize(40, 10)
	m.cursor = 5

	prompts := []model.Note{
		{Name: "20260301-100000"},
		{Name: "20260201-100000"},
	}
	m = m.SetPrompts(prompts)

	if m.cursor != 1 {
		t.Errorf("cursor should be clamped to %d, got %d", 1, m.cursor)
	}
	if len(m.items) != 2 {
		t.Errorf("expected 2 prompts, got %d", len(m.items))
	}
}
