package tui

import (
	"testing"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
)

func TestTodosScrollDown(t *testing.T) {
	// Create 10 open todos in a small viewport.
	// Each open todo after the first takes 2 visible lines (blank + text),
	// so 10 items need ~19 lines. With height=6, only ~3-4 items fit.
	var todos []model.Todo
	for i := 0; i < 10; i++ {
		todos = append(todos, model.Todo{Text: "task"})
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 6)
	m = m.SetFocus(true)

	// Navigate to item 5 — well past visible area
	for i := 0; i < 5; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if m.cursor != 5 {
		t.Fatalf("cursor = %d, want 5", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced past 0 for cursor=5 in height=6, got %d", m.scrollOffset)
	}
}

func TestTodosScrollDownWithDone(t *testing.T) {
	// Mix of open and done todos. The hint/divider section between
	// open and done takes ~5 extra lines, making scroll even more important.
	todos := []model.Todo{
		{Text: "open 1"},
		{Text: "open 2"},
		{Text: "open 3"},
		{Text: "done 1", Done: true},
		{Text: "done 2", Done: true},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 8)
	m = m.SetFocus(true)

	// Navigate to first done item (index 3)
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if m.cursor != 3 {
		t.Fatalf("cursor = %d, want 3", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced for cursor=3 with hint/divider in height=8, got %d", m.scrollOffset)
	}
}

func TestTodosCopyYank(t *testing.T) {
	if clipboard.Unsupported {
		t.Skip("clipboard not available in this environment")
	}

	todos := []model.Todo{
		{Text: "first todo"},
		{Text: "second todo"},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Yank the first todo
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if m.statusMsg != "Copied!" {
		t.Errorf("statusMsg = %q, want %q", m.statusMsg, "Copied!")
	}
	if cmd == nil {
		t.Fatal("expected a tick command for clearing status message")
	}

	got, err := clipboard.ReadAll()
	if err != nil {
		t.Fatalf("clipboard.ReadAll: %v", err)
	}
	if got != "first todo" {
		t.Errorf("clipboard = %q, want %q", got, "first todo")
	}

	// Move to second todo and yank
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	got, _ = clipboard.ReadAll()
	if got != "second todo" {
		t.Errorf("clipboard = %q, want %q", got, "second todo")
	}
}

func TestTodosCopyEmptyList(t *testing.T) {
	m := newTodos(nil, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if m.statusMsg != "" {
		t.Errorf("statusMsg = %q, want empty for empty list", m.statusMsg)
	}
	if cmd != nil {
		t.Error("expected no command for yank on empty list")
	}
}

func TestTodosClearStatusMsg(t *testing.T) {
	todos := []model.Todo{{Text: "a todo"}}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 10)
	m.statusMsg = "Copied!"

	m, _ = m.Update(clearStatusMsg{})

	if m.statusMsg != "" {
		t.Errorf("statusMsg = %q, want empty after clearStatusMsg", m.statusMsg)
	}
}

func TestTodosScrollUpAfterDown(t *testing.T) {
	var todos []model.Todo
	for i := 0; i < 10; i++ {
		todos = append(todos, model.Todo{Text: "task"})
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 6)
	m = m.SetFocus(true)

	// Scroll down
	for i := 0; i < 8; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	savedOffset := m.scrollOffset

	// Scroll back up
	for i := 0; i < 8; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	}

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 after scrolling back to top", m.scrollOffset)
	}
	if savedOffset == 0 {
		t.Errorf("should have scrolled down before scrolling back up")
	}
}
