package tui

import (
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func tempTodosStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

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
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if m.cursor != 5 {
		t.Fatalf("cursor = %d, want 5", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced past 0 for cursor=5 in height=6, got %d", m.scrollOffset)
	}
}

func TestTodosScrollDownWithDone(t *testing.T) {
	// Mix of open and done todos. Done items are not navigable in
	// the default (pending) view — cursor stops at the last open item.
	// The done section + divider still occupy visual lines, so we verify
	// scrolling advances to keep the cursor visible.
	todos := []model.Todo{
		{Text: "open 1"},
		{Text: "open 2"},
		{Text: "open 3"},
		{Text: "done 1", Status: model.StatusDone},
		{Text: "done 2", Status: model.StatusDone},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 4) // small viewport to force scrolling
	m = m.SetFocus(true)

	// Navigate to last open item (index 2); done items are skipped.
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced for cursor=2 in height=4, got %d", m.scrollOffset)
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
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
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
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	savedOffset := m.scrollOffset

	// Scroll back up
	for i := 0; i < 8; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
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
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if m.cursor != 4 {
		t.Fatalf("cursor = %d, want 4", m.cursor)
	}
	if m.scrollOffset == 0 {
		t.Errorf("scrollOffset should have advanced for wrapped items in small viewport, got %d", m.scrollOffset)
	}
}

func TestTodosDeleteConfirmCancel(t *testing.T) {
	todos := []model.Todo{{Text: "keep me"}}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press x — should enter confirmation mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.confirm != confirmDelete {
		t.Fatal("expected confirm == confirmDelete after 'x'")
	}

	// View should show confirmation text
	view := m.View()
	if !strings.Contains(view, "Delete todo?") {
		t.Errorf("view should show delete confirmation, got %q", view)
	}

	// Press 'n' to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.confirm != confirmNone {
		t.Error("confirmation should be cancelled")
	}
	if len(m.todos) != 1 {
		t.Errorf("todo should not be deleted, got %d todos", len(m.todos))
	}
}

func TestTodosDeleteConfirm(t *testing.T) {
	s := tempTodosStore(t)
	todos := []model.Todo{{Text: "delete me"}, {Text: "keep me"}}

	m := newTodos(todos, s)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press x then y to confirm delete
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.confirm != confirmNone {
		t.Error("confirm should be confirmNone after confirmation")
	}
	if len(m.todos) != 1 {
		t.Errorf("expected 1 todo after delete, got %d", len(m.todos))
	}
	if m.todos[0].Text != "keep me" {
		t.Errorf("wrong todo remaining: %q", m.todos[0].Text)
	}
}

func TestTodosDeleteEmptyList(t *testing.T) {
	m := newTodos(nil, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press x on empty list — should not enter confirmation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.confirm != confirmNone {
		t.Error("should not enter confirmation on empty list")
	}
}

func TestTodosPurgeConfirmCancel(t *testing.T) {
	todos := []model.Todo{
		{Text: "open"},
		{Text: "done", Status: model.StatusDone},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press X — should enter purge confirmation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if m.confirm != confirmPurge {
		t.Fatal("expected confirm == confirmPurge after 'X'")
	}

	view := m.View()
	if !strings.Contains(view, "Clear all open todos?") {
		t.Errorf("view should show purge confirmation, got %q", view)
	}

	// Press esc to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirm != confirmNone {
		t.Error("purge confirmation should be cancelled")
	}
	if len(m.todos) != 2 {
		t.Errorf("todos should not be purged, got %d", len(m.todos))
	}
}

func TestTodosPurgeConfirm(t *testing.T) {
	s := tempTodosStore(t)
	todos := []model.Todo{
		{Text: "open"},
		{Text: "done1", Status: model.StatusDone},
		{Text: "done2", Status: model.StatusDone},
	}

	m := newTodos(todos, s)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Press X then y to confirm purge of open todos
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.confirm != confirmNone {
		t.Error("confirm should be confirmNone after confirmation")
	}
	if len(m.todos) != 2 {
		t.Errorf("expected 2 done todos after purge, got %d", len(m.todos))
	}
	if m.todos[0].Text != "done1" {
		t.Errorf("wrong todo remaining: %q", m.todos[0].Text)
	}
}

func TestMoveTodoDown(t *testing.T) {
	s := tempTodosStore(t)
	todos := []model.Todo{
		{Text: "first"},
		{Text: "second"},
		{Text: "third"},
	}

	m := newTodos(todos, s)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Cursor at 0 ("first"), press J to move it down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})

	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (cursor follows moved item)", m.cursor)
	}
	if m.todos[0].Text != "second" || m.todos[1].Text != "first" {
		t.Errorf("todos = [%q, %q, ...], want [second, first, ...]", m.todos[0].Text, m.todos[1].Text)
	}

	// Verify persisted
	loaded, err := s.LoadTodos()
	if err != nil {
		t.Fatalf("LoadTodos: %v", err)
	}
	if loaded[0].Text != "second" || loaded[1].Text != "first" {
		t.Errorf("persisted order wrong: [%q, %q, ...]", loaded[0].Text, loaded[1].Text)
	}
}

func TestMoveTodoUp(t *testing.T) {
	s := tempTodosStore(t)
	todos := []model.Todo{
		{Text: "first"},
		{Text: "second"},
		{Text: "third"},
	}

	m := newTodos(todos, s)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Move cursor to "third" (index 2), then press K to move up
	m = m.moveCursor(2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})

	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
	if m.todos[1].Text != "third" || m.todos[2].Text != "second" {
		t.Errorf("todos = [..., %q, %q], want [..., third, second]", m.todos[1].Text, m.todos[2].Text)
	}
}

func TestMoveTodoNoopAtStatusBoundary(t *testing.T) {
	todos := []model.Todo{
		{Text: "in progress", Status: model.StatusInProgress},
		{Text: "open one"},
		{Text: "open two"},
		{Text: "done one", Status: model.StatusDone},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Cursor at 0 (in-progress), J should be no-op (next item is open)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (no-op at boundary)", m.cursor)
	}
	if m.todos[0].Text != "in progress" {
		t.Errorf("item should not have moved: %q", m.todos[0].Text)
	}

	// Cursor at 1 (first open), K should be no-op (prev item is in-progress)
	m = m.moveCursor(1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (no-op at boundary)", m.cursor)
	}

	// Cursor at 2 (last open), J should be no-op (next item is done)
	m = m.moveCursor(1) // now at 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (no-op at boundary)", m.cursor)
	}
}

func TestMoveTodoNoopAtListEdges(t *testing.T) {
	todos := []model.Todo{
		{Text: "only open one"},
		{Text: "only open two"},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// K at top of list — no-op
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}

	// J at bottom of list — no-op
	m = m.moveCursor(1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestMoveTodoEmptyList(t *testing.T) {
	m := newTodos(nil, nil)
	m = m.SetSize(40, 10)
	m = m.SetFocus(true)

	// Should not panic on empty list
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})

	if len(m.todos) != 0 {
		t.Errorf("expected empty list")
	}
}

func TestToggleShowCompleted(t *testing.T) {
	todos := []model.Todo{
		{Text: "open one"},
		{Text: "open two"},
		{Text: "done one", Status: model.StatusDone},
		{Text: "done two", Status: model.StatusDone},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Default: pending view — done items hidden.
	view := m.View()
	if !strings.Contains(view, "open one") {
		t.Error("pending view should show open items")
	}
	if strings.Contains(view, "done one") {
		t.Error("pending view should hide done items")
	}

	// Press 'v' to toggle to completed view.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.showCompleted {
		t.Fatal("showCompleted should be true after pressing v")
	}
	// Cursor lands on first visible done item (index 2 after sorting: open, open, done, done).
	if m.cursor != 2 {
		t.Errorf("cursor should be on first done item (index 2), got %d", m.cursor)
	}

	view = m.View()
	if strings.Contains(view, "open one") {
		t.Error("completed view should hide open items")
	}
	if !strings.Contains(view, "done one") {
		t.Error("completed view should show done items")
	}
	if strings.Contains(view, "Add Todo") {
		t.Error("completed view should not show Add Todo hint")
	}

	// Press 'v' again to go back to pending view.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if m.showCompleted {
		t.Fatal("showCompleted should be false after second v press")
	}
	view = m.View()
	if !strings.Contains(view, "open one") {
		t.Error("should be back to pending view")
	}
}

func TestToggleShowCompletedAddNoOp(t *testing.T) {
	todos := []model.Todo{
		{Text: "done", Status: model.StatusDone},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Switch to completed view
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	// Press 'a' — should be no-op
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.inputActive {
		t.Error("'a' should be no-op in completed view")
	}
	if cmd != nil {
		t.Error("no command expected from no-op 'a'")
	}
}

func TestToggleShowCompletedEmptyState(t *testing.T) {
	todos := []model.Todo{
		{Text: "open"},
	}

	m := newTodos(todos, nil)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Switch to completed view — no done items
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	view := m.View()
	if !strings.Contains(view, "No completed todos") {
		t.Errorf("expected empty state message, got %q", view)
	}
}

func TestPurgeSwitchesBackToPendingView(t *testing.T) {
	s := tempTodosStore(t)
	todos := []model.Todo{
		{Text: "open"},
		{Text: "done", Status: model.StatusDone},
	}

	m := newTodos(todos, s)
	m = m.SetSize(40, 20)
	m = m.SetFocus(true)

	// Switch to completed view
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.showCompleted {
		t.Fatal("should be in completed view")
	}

	// Purge done items
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.showCompleted {
		t.Error("should auto-switch back to pending view after purge")
	}
	view := m.View()
	if !strings.Contains(view, "open") {
		t.Error("should show open items after purge switches back")
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
