package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func testTemplateStore(t *testing.T) *store.TemplateStore {
	t.Helper()
	return store.NewTemplateStore(filepath.Join(t.TempDir(), "templates"))
}

func setupModalWithTemplates(t *testing.T, names ...string) templateModal {
	t.Helper()
	ts := testTemplateStore(t)
	for _, name := range names {
		ts.SaveTemplate(name, []model.Todo{{Text: "item from " + name}})
	}
	m := newTemplateModal(ts)
	m = m.open(false, 80, 24)
	return m
}

func TestTemplateBrowseNavigation(t *testing.T) {
	m := setupModalWithTemplates(t, "alpha", "beta", "gamma")

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}

	// Move down again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("after down: cursor = %d, want 2", m.cursor)
	}

	// Wrap around
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 0 {
		t.Errorf("after wrap: cursor = %d, want 0", m.cursor)
	}

	// Wrap up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 2 {
		t.Errorf("after up wrap: cursor = %d, want 2", m.cursor)
	}
}

func TestTemplateBrowseSelectImports(t *testing.T) {
	m := setupModalWithTemplates(t, "workflow")

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter, got nil")
	}

	msg := cmd()
	imp, ok := msg.(importTemplateMsg)
	if !ok {
		t.Fatalf("expected importTemplateMsg, got %T", msg)
	}
	if len(imp.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(imp.todos))
	}
	if imp.todos[0].Status != model.StatusOpen {
		t.Errorf("imported todo should be StatusOpen, got %d", imp.todos[0].Status)
	}
}

func TestTemplateBrowseEscCancels(t *testing.T) {
	m := setupModalWithTemplates(t, "workflow")

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command on esc, got nil")
	}

	msg := cmd()
	if _, ok := msg.(exitTemplateMsg); !ok {
		t.Fatalf("expected exitTemplateMsg, got %T", msg)
	}
}

func TestTemplateBrowseEmptyList(t *testing.T) {
	ts := testTemplateStore(t)
	m := newTemplateModal(ts)
	m = m.open(false, 80, 24)

	if len(m.templates) != 0 {
		t.Fatalf("expected empty template list, got %d", len(m.templates))
	}

	// Enter on empty list should do nothing
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter on empty list should produce no command")
	}

	// Esc still works
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should still work on empty list")
	}
}

func TestTemplateSaveFlow(t *testing.T) {
	ts := testTemplateStore(t)
	m := newTemplateModal(ts)
	m = m.open(true, 80, 24)

	if m.mode != tmNameInput {
		t.Fatalf("expected tmNameInput mode for save-as, got %d", m.mode)
	}

	// Type a name
	m.input.SetValue("mytemplate")

	// Confirm
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter, got nil")
	}

	msg := cmd()
	save, ok := msg.(saveTemplateMsg)
	if !ok {
		t.Fatalf("expected saveTemplateMsg, got %T", msg)
	}
	if save.name != "mytemplate" {
		t.Errorf("save.name = %q, want %q", save.name, "mytemplate")
	}
}

func TestTemplateSaveEmptyNameIgnored(t *testing.T) {
	ts := testTemplateStore(t)
	m := newTemplateModal(ts)
	m = m.open(true, 80, 24)

	m.input.SetValue("")

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with empty name should produce no command")
	}
}

func TestTemplateSaveOverwriteConfirmation(t *testing.T) {
	ts := testTemplateStore(t)
	ts.SaveTemplate("existing", []model.Todo{{Text: "old"}})

	m := newTemplateModal(ts)
	m = m.open(true, 80, 24)
	m.input.SetValue("existing")

	// Enter should trigger confirm mode
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("should not emit command yet, should show confirmation")
	}
	if m.mode != tmConfirmOverwrite {
		t.Fatalf("expected tmConfirmOverwrite, got %d", m.mode)
	}

	// Press 'y' to confirm
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected command after confirming overwrite")
	}

	msg := cmd()
	save, ok := msg.(saveTemplateMsg)
	if !ok {
		t.Fatalf("expected saveTemplateMsg, got %T", msg)
	}
	if save.name != "existing" {
		t.Errorf("save.name = %q, want %q", save.name, "existing")
	}
}

func TestTemplateSaveOverwriteCancel(t *testing.T) {
	ts := testTemplateStore(t)
	ts.SaveTemplate("existing", []model.Todo{{Text: "old"}})

	m := newTemplateModal(ts)
	m = m.open(true, 80, 24)
	m.input.SetValue("existing")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != tmConfirmOverwrite {
		t.Fatalf("expected tmConfirmOverwrite, got %d", m.mode)
	}

	// Press 'n' to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.mode != tmNameInput {
		t.Errorf("expected back to tmNameInput after cancel, got %d", m.mode)
	}
}

func TestTemplateFooterHints(t *testing.T) {
	ts := testTemplateStore(t)
	m := newTemplateModal(ts)

	// Browse with no templates
	m = m.open(false, 80, 24)
	if hint := m.FooterHint(); hint != "esc close" {
		t.Errorf("empty browse hint = %q, want %q", hint, "esc close")
	}

	// Browse with templates
	ts.SaveTemplate("x", []model.Todo{{Text: "a"}})
	m = m.open(false, 80, 24)
	if hint := m.FooterHint(); hint != "↑/↓ navigate · enter select · esc cancel" {
		t.Errorf("browse hint = %q", hint)
	}

	// Name input
	m = m.open(true, 80, 24)
	if hint := m.FooterHint(); hint != "enter confirm · esc cancel" {
		t.Errorf("name input hint = %q", hint)
	}

	// Confirm overwrite
	m.mode = tmConfirmOverwrite
	if hint := m.FooterHint(); hint != "overwrite? y/n" {
		t.Errorf("overwrite hint = %q", hint)
	}
}

func TestTemplateViewZeroDimensions(t *testing.T) {
	ts := testTemplateStore(t)
	m := newTemplateModal(ts)
	// Don't call open (width/height = 0)
	if v := m.View(); v != "" {
		t.Errorf("expected empty view with zero dimensions, got %q", v)
	}
}

func TestTemplateImportResetsTodoStatus(t *testing.T) {
	ts := testTemplateStore(t)
	// Save a template with mixed statuses
	ts.SaveTemplate("mixed", []model.Todo{
		{Text: "open", Status: model.StatusOpen},
		{Text: "done", Status: model.StatusDone},
		{Text: "wip", Status: model.StatusInProgress},
	})

	m := newTemplateModal(ts)
	m = m.open(false, 80, 24)

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command")
	}

	msg := cmd()
	imp := msg.(importTemplateMsg)
	for i, todo := range imp.todos {
		if todo.Status != model.StatusOpen {
			t.Errorf("todo[%d] status = %d, want StatusOpen (0)", i, todo.Status)
		}
	}
}
