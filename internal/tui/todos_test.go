package tui

import (
	"testing"

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
