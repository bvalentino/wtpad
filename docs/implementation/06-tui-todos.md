# 06 — Todo Pane

**State:** `done`

**Depends on:** `05-tui-root.md`
**Blocks:** `11-resize.md`

---

## Goal

Implement the todo pane in `internal/tui/todos.go`. This handles rendering the todo list and all todo interactions: navigation, add, toggle done, edit, delete.

## Tasks

### Rendering
- [x] Render open todos first, then completed todos (dimmed, strikethrough via lipgloss)
- [x] Highlight the selected item with a distinct background or indicator
- [x] Show `○` prefix for open todos, `✓` for done
- [x] Scroll the list if it overflows the pane height

### Navigation
- [x] `j` / `↓` — move selection down
- [x] `k` / `↑` — move selection up
- [x] Clamp selection to valid range

### Add Todo
- [x] `a` — switch root mode to `modeInput`, render a text input at the bottom of the pane (use `bubbles/textinput`)
- [x] `Enter` in input mode — append todo to list, call `store.SaveTodos()`, return to `modeNormal`
- [x] `Esc` in input mode — discard, return to `modeNormal`

### Toggle Done
- [x] `d` or `Space` — toggle `Done` on selected item, call `store.SaveTodos()`, save

### Edit
- [x] `Enter` in normal mode — populate textinput with current todo text, switch to `modeInput`
- [x] Saving an edit updates the existing todo's `Text` in place (does not create a new one)

### Delete
- [x] `x` or `Delete` — remove selected todo from slice, call `store.SaveTodos()`, adjust selection index
- [x] `D` — remove all todos where `Done == true`, call `store.SaveTodos()`

## Acceptance

- Full list renders correctly with open/done distinction
- All keybindings work and persist to disk
- Adding, editing, completing, and deleting todos all survive a restart

## Notes

- Added `charmbracelet/bubbles` dependency for `textinput` component
- `todosModel` owns `[]model.Todo` exclusively; root App does not keep a copy
- All helpers use value receivers returning modified model (consistent with Bubble Tea convention)
- `SetSize`/`SetFocus` also use value receivers returning the model
- Mode transitions use `enterInputMsg`/`exitInputMsg` messages; root suppresses global keys (q/Tab) in `modeInput`
- `sortTodos()` stable-partitions open items first, done items last
- Scroll offset managed in `adjustScroll()` helper called from mutation points; `View()` is a pure read
- `save()` helper centralizes `store.SaveTodos()` calls with error logging
- `ctrl+c` handled unconditionally (always quits); `q` gated behind `modeNormal`
