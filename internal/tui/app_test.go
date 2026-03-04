package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func tempStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

func tempTemplateStoreForApp(t *testing.T) *store.TemplateStore {
	t.Helper()
	return store.NewTemplateStore(filepath.Join(t.TempDir(), "templates"))
}

func tempPromptStoreForApp(t *testing.T) *store.PromptStore {
	t.Helper()
	return store.NewPromptStore(filepath.Join(t.TempDir(), "prompts"))
}

func TestViewBeforeWindowSizeMsg(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), []model.Todo{{Text: "task"}}, nil, nil, "main")

	// View() is called before any WindowSizeMsg, so width/height are 0.
	// This must not panic — returns empty string gracefully.
	out := app.View()
	if out != "" {
		t.Errorf("expected empty output before WindowSizeMsg, got %q", out)
	}
}

func sendResize(t *testing.T, m tea.Model, w, h int) App {
	t.Helper()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(App)
}

func TestResizePropagatesBothPanes(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), []model.Todo{{Text: "task"}}, nil, nil, "main")

	app = sendResize(t, app, 80, 40)

	if app.todosPane.width == 0 || app.todosPane.height == 0 {
		t.Error("todosPane dimensions not set after resize")
	}
	if app.notesPane.width == 0 || app.notesPane.height == 0 {
		t.Error("notesPane dimensions not set after resize")
	}
	// Both panes should have the same dimensions
	if app.todosPane.width != app.notesPane.width {
		t.Errorf("pane widths differ: todos=%d notes=%d", app.todosPane.width, app.notesPane.width)
	}
	if app.todosPane.height != app.notesPane.height {
		t.Errorf("pane heights differ: todos=%d notes=%d", app.todosPane.height, app.notesPane.height)
	}
}

func TestResizeSmallTerminal(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), []model.Todo{{Text: "task"}}, nil, nil, "main")

	// Very small terminal — must not panic
	app = sendResize(t, app, 10, 5)

	if app.contentHeight < 1 {
		t.Errorf("contentHeight should be >= 1, got %d", app.contentHeight)
	}
	if app.contentWidth < 1 {
		t.Errorf("contentWidth should be >= 1, got %d", app.contentWidth)
	}

	// View should produce output without panicking
	out := app.View()
	if out == "" {
		t.Error("expected non-empty output for small terminal")
	}
}

func TestResizeHeaderToggle(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	// Tall terminal: full ASCII header
	app = sendResize(t, app, 80, 40)
	if !app.showFullHeader {
		t.Error("expected full header for height >= 30")
	}
	if app.headerHeight != asciiHeaderHeight {
		t.Errorf("headerHeight = %d, want %d", app.headerHeight, asciiHeaderHeight)
	}

	// Short terminal: compact header
	app = sendResize(t, app, 80, 20)
	if app.showFullHeader {
		t.Error("expected compact header for height < 30")
	}
	if app.headerHeight != 1 {
		t.Errorf("headerHeight = %d, want 1", app.headerHeight)
	}
}

func TestRapidResize(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{{Text: "task 1"}, {Text: "task 2"}}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")

	// Simulate rapid resize — no panics, valid state after each
	sizes := [][2]int{
		{80, 40}, {120, 50}, {40, 15}, {200, 60}, {10, 5}, {80, 30},
	}
	for _, sz := range sizes {
		app = sendResize(t, app, sz[0], sz[1])
		// View must not panic
		out := app.View()
		if out == "" && sz[0] > 0 && sz[1] > 0 {
			t.Errorf("empty view for size %dx%d", sz[0], sz[1])
		}
	}
}

func TestResizeInEditorMode(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	// Set initial size, then enter editor
	app = sendResize(t, app, 80, 40)
	updated, _ := app.Update(enterEditorMsg{name: "", body: "hello"})
	app = updated.(App)

	if app.mode != modeEditor {
		t.Fatalf("expected modeEditor, got %d", app.mode)
	}

	// Resize while in editor mode — editor gets full terminal dimensions (full-screen)
	app = sendResize(t, app, 120, 50)

	if app.editorPane.width != 120 {
		t.Errorf("editor width = %d, want 120", app.editorPane.width)
	}
	if app.editorPane.height != 50 {
		t.Errorf("editor height = %d, want 50", app.editorPane.height)
	}

	// View must not panic
	out := app.View()
	if out == "" {
		t.Error("expected non-empty editor view after resize")
	}
}

func TestResizeInHelpMode(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	// Set initial size
	app = sendResize(t, app, 80, 40)

	// Enter help mode (? returns a command that emits enterHelpMsg)
	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app = updated.(App)
	if cmd == nil {
		t.Fatal("? should produce a command")
	}
	updated, _ = app.Update(cmd())
	app = updated.(App)
	if app.mode != modeHelp {
		t.Fatalf("expected modeHelp, got %d", app.mode)
	}

	// Resize while in help mode
	app = sendResize(t, app, 120, 50)

	if app.helpPane.width != 120 {
		t.Errorf("help width = %d, want 120", app.helpPane.width)
	}
	if app.helpPane.height != 50 {
		t.Errorf("help height = %d, want 50", app.helpPane.height)
	}

	// View must not panic and should contain help content
	out := app.View()
	if !strings.Contains(out, "keyboard shortcuts") {
		t.Error("help view should contain 'keyboard shortcuts' after resize")
	}
}

func TestEditorRendersFullScreen(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	app = sendResize(t, app, 80, 40)
	updated, _ := app.Update(enterEditorMsg{name: "", body: "hello"})
	app = updated.(App)

	out := app.View()

	// Full-screen editor should NOT show tab strip
	if strings.Contains(out, "Todo") {
		t.Error("full-screen editor should not contain tab strip")
	}

	// Should show bordered box with top and bottom borders
	if !strings.Contains(out, "╭") {
		t.Error("editor view should contain top border")
	}
	if !strings.Contains(out, "╰") {
		t.Error("editor view should contain bottom border")
	}

	// Should show footer with editor hints
	if !strings.Contains(out, "ctrl+s save") {
		t.Error("editor view should contain footer hints")
	}

	// Should show "New Note" in the top border (name is empty = new note)
	if !strings.Contains(out, "New Note") {
		t.Error("editor view should contain 'New Note' title")
	}
}

func TestEditorFooterShowsContextualHints(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	app = sendResize(t, app, 80, 40)
	updated, _ := app.Update(enterEditorMsg{name: "", body: "hello"})
	app = updated.(App)

	// Type something to make it dirty, then press Esc to trigger confirm
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	app = updated.(App)
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(App)

	out := app.View()
	if !strings.Contains(out, "discard changes? y/n") {
		t.Error("editor footer should show discard confirmation, got: " + out[len(out)-100:])
	}
}

func TestEditorDimensionsMatchTerminal(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	app = sendResize(t, app, 80, 40)
	updated, _ := app.Update(enterEditorMsg{name: "", body: "test"})
	app = updated.(App)

	// Editor should receive full terminal dimensions (full-screen)
	if app.editorPane.width != 80 {
		t.Errorf("editor width = %d, want 80", app.editorPane.width)
	}
	if app.editorPane.height != 40 {
		t.Errorf("editor height = %d, want 40", app.editorPane.height)
	}
}

func TestToggleInProgress(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{{Text: "task 1"}, {Text: "task 2"}}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")
	app = sendResize(t, app, 80, 40)

	// Press 'p' to toggle first todo to in-progress
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	app = updated.(App)

	// First open item should now be in-progress (and sorted after remaining open items)
	found := false
	for _, todo := range app.todosPane.todos {
		if todo.Text == "task 1" {
			if todo.Status != model.StatusInProgress {
				t.Errorf("expected task 1 to be in-progress, got status %d", todo.Status)
			}
			found = true
		}
	}
	if !found {
		t.Error("task 1 not found in todos")
	}
}

func TestToggleInProgressOnDoneIsNoOp(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{{Text: "done task", Status: model.StatusDone}}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")
	app = sendResize(t, app, 80, 40)

	// Press 'p' on a done item — should be a no-op
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	app = updated.(App)

	if app.todosPane.todos[0].Status != model.StatusDone {
		t.Error("pressing p on a done todo should not change status")
	}
}

func TestToggleDoneClearsInProgress(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{{Text: "wip task", Status: model.StatusInProgress}}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")
	app = sendResize(t, app, 80, 40)

	// Press space to mark in-progress item as done
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	app = updated.(App)

	if app.todosPane.todos[0].Status != model.StatusDone {
		t.Errorf("expected StatusDone after pressing d, got %d", app.todosPane.todos[0].Status)
	}
}

func TestSortTodosThreeGroups(t *testing.T) {
	todos := []model.Todo{
		{Text: "done", Status: model.StatusDone},
		{Text: "wip", Status: model.StatusInProgress},
		{Text: "open"},
	}

	sorted := sortTodos(todos)

	if sorted[0].Text != "wip" {
		t.Errorf("sorted[0] = %q, want 'wip'", sorted[0].Text)
	}
	if sorted[1].Text != "open" {
		t.Errorf("sorted[1] = %q, want 'open'", sorted[1].Text)
	}
	if sorted[2].Text != "done" {
		t.Errorf("sorted[2] = %q, want 'done'", sorted[2].Text)
	}
}

func TestViewRendersInProgressPrefix(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{
		{Text: "open task"},
		{Text: "wip task", Status: model.StatusInProgress},
		{Text: "done task", Status: model.StatusDone},
	}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")
	app = sendResize(t, app, 80, 40)

	out := app.View()

	// Default view shows open and in-progress, hides done.
	if !strings.Contains(out, "○ open task") {
		t.Error("expected '○ open task' in view")
	}
	if !strings.Contains(out, "▸ wip task") {
		t.Error("expected '▸ wip task' in view")
	}
	if strings.Contains(out, "✓ done task") {
		t.Error("done task should be hidden in default (pending) view")
	}

	// Press 'v' to toggle to completed view.
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	app = updated.(App)
	out = app.View()

	if !strings.Contains(out, "✓ done task") {
		t.Error("expected '✓ done task' in completed view")
	}
	if strings.Contains(out, "○ open task") {
		t.Error("open task should be hidden in completed view")
	}
}

func TestFooterCountsWithInProgress(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{
		{Text: "open"},
		{Text: "wip", Status: model.StatusInProgress},
		{Text: "done", Status: model.StatusDone},
	}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")
	app = sendResize(t, app, 80, 40)

	out := app.View()

	if !strings.Contains(out, "1 open") {
		t.Error("expected '1 open' in footer")
	}
	if !strings.Contains(out, "1 in progress") {
		t.Error("expected '1 in progress' in footer")
	}
	if !strings.Contains(out, "1 done") {
		t.Error("expected '1 done' in footer")
	}
}

func TestFooterOmitsInProgressWhenZero(t *testing.T) {
	s := tempStore(t)
	todos := []model.Todo{
		{Text: "open"},
		{Text: "done", Status: model.StatusDone},
	}
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), todos, nil, nil, "main")
	app = sendResize(t, app, 80, 40)

	out := app.View()

	if strings.Contains(out, "in progress") {
		t.Error("footer should not show 'in progress' when count is 0")
	}
}

func TestResizeContentDimensions(t *testing.T) {
	s := tempStore(t)
	app := New(s, tempTemplateStoreForApp(t), tempPromptStoreForApp(t), nil, nil, nil, "main")

	app = sendResize(t, app, 80, 40)

	// With height >= 30, full header is shown (6 lines)
	// contentHeight = 40 - 6 (header) - 3 (tabs) - 1 (footer) - 1 (border) = 29
	expectedHeight := 40 - asciiHeaderHeight - tabStripHeight - footerHeight - 1
	if app.contentHeight != expectedHeight {
		t.Errorf("contentHeight = %d, want %d", app.contentHeight, expectedHeight)
	}

	// contentWidth = 80 - 2 (side borders) - 2 (spacing) = 76
	expectedWidth := 80 - sideBorderSize - 2
	if app.contentWidth != expectedWidth {
		t.Errorf("contentWidth = %d, want %d", app.contentWidth, expectedWidth)
	}
}
