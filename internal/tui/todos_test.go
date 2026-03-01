package tui

import (
	"strings"
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

func TestWrapTextNoWrap(t *testing.T) {
	lines := wrapText("short", 20)
	if len(lines) != 1 || lines[0] != "short" {
		t.Errorf("wrapText('short', 20) = %v, want ['short']", lines)
	}
}

func TestWrapTextExactFit(t *testing.T) {
	lines := wrapText("12345", 5)
	if len(lines) != 1 || lines[0] != "12345" {
		t.Errorf("wrapText('12345', 5) = %v, want ['12345']", lines)
	}
}

func TestWrapTextMultiWord(t *testing.T) {
	lines := wrapText("hello world foo", 11)
	// "hello world" = 11 fits, "foo" goes to next line
	if len(lines) != 2 {
		t.Fatalf("wrapText got %d lines, want 2: %v", len(lines), lines)
	}
	if lines[0] != "hello world" {
		t.Errorf("line[0] = %q, want %q", lines[0], "hello world")
	}
	if lines[1] != "foo" {
		t.Errorf("line[1] = %q, want %q", lines[1], "foo")
	}
}

func TestWrapTextLongWord(t *testing.T) {
	lines := wrapText("abcdefghij", 4)
	// Hard-break: "abcd", "efgh", "ij"
	if len(lines) != 3 {
		t.Fatalf("wrapText got %d lines, want 3: %v", len(lines), lines)
	}
	if lines[0] != "abcd" {
		t.Errorf("line[0] = %q, want %q", lines[0], "abcd")
	}
	if lines[1] != "efgh" {
		t.Errorf("line[1] = %q, want %q", lines[1], "efgh")
	}
	if lines[2] != "ij" {
		t.Errorf("line[2] = %q, want %q", lines[2], "ij")
	}
}

func TestWrapTextZeroWidth(t *testing.T) {
	lines := wrapText("hello", 0)
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("wrapText('hello', 0) = %v, want ['']", lines)
	}
}

func TestWrapTextLineCount(t *testing.T) {
	if c := len(wrapText("short", 40)); c != 1 {
		t.Errorf("wrapText line count for short text = %d, want 1", c)
	}

	long := "this is a longer todo that should wrap across multiple lines"
	if c := len(wrapText(long, 20)); c <= 1 {
		t.Errorf("wrapText line count for long text at width 20 = %d, want > 1", c)
	}
}

func TestTodosScrollDownWithWrappedItems(t *testing.T) {
	// Create todos where some have long text that wraps.
	todos := []model.Todo{
		{Text: "short"},
		{Text: "this is a really long todo item that should wrap to multiple lines in a narrow viewport"},
		{Text: "another long todo item that definitely wraps when the width is small"},
		{Text: "short2"},
		{Text: "short3"},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(30, 6)
	m = m.SetFocus(true)

	// Navigate down past the wrapped items
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if m.cursor != 4 {
		t.Fatalf("cursor = %d, want 4", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced for wrapped items in small viewport, got %d", m.scrollOffset)
	}
}

func TestViewShowsFullWrappedText(t *testing.T) {
	longText := "buy groceries including milk eggs bread butter and cheese from the store"
	todos := []model.Todo{{Text: longText}}

	m := newTodos(todos, nil)
	m = m.SetSize(30, 10)
	m = m.SetFocus(true)

	output := m.View()

	// The full text should appear across wrapped lines (no ellipsis).
	if strings.Contains(output, "…") {
		t.Error("View() output contains ellipsis; text should be wrapped, not truncated")
	}

	// All words from the original text should be present.
	for _, word := range strings.Fields(longText) {
		if !strings.Contains(output, word) {
			t.Errorf("View() output missing word %q from wrapped todo", word)
		}
	}
}
