package tui

import (
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

func TestViewBeforeWindowSizeMsg(t *testing.T) {
	s := tempStore(t)
	app := New(s, []model.Todo{{Text: "task"}}, nil, "main")

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
	app := New(s, []model.Todo{{Text: "task"}}, nil, "main")

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
	app := New(s, []model.Todo{{Text: "task"}}, nil, "main")

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
	app := New(s, nil, nil, "main")

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
	app := New(s, todos, nil, "main")

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
	app := New(s, nil, nil, "main")

	// Set initial size, then enter editor
	app = sendResize(t, app, 80, 40)
	updated, _ := app.Update(enterEditorMsg{name: "", body: "hello"})
	app = updated.(App)

	if app.mode != modeEditor {
		t.Fatalf("expected modeEditor, got %d", app.mode)
	}

	// Resize while in editor mode
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
	app := New(s, nil, nil, "main")

	// Set initial size
	app = sendResize(t, app, 80, 40)

	// Enter help mode
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
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

func TestResizeContentDimensions(t *testing.T) {
	s := tempStore(t)
	app := New(s, nil, nil, "main")

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
