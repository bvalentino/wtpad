# 06 — Todo Pane

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `05-tui-root.md`
**Blocks:** `11-resize.md`

---

## Goal

Implement the todo pane in `internal/tui/todos.go`. This handles rendering the todo list and all todo interactions: navigation, add, toggle done, edit, delete.

## Tasks

### Rendering
- [ ] Render open todos first, then completed todos (dimmed, strikethrough via lipgloss)
- [ ] Highlight the selected item with a distinct background or indicator
- [ ] Show `○` prefix for open todos, `✓` for done
- [ ] Scroll the list if it overflows the pane height

### Navigation
- [ ] `j` / `↓` — move selection down
- [ ] `k` / `↑` — move selection up
- [ ] Clamp selection to valid range

### Add Todo
- [ ] `a` — switch root mode to `modeInput`, render a text input at the bottom of the pane (use `bubbles/textinput`)
- [ ] `Enter` in input mode — append todo to list, call `store.SaveTodos()`, return to `modeNormal`
- [ ] `Esc` in input mode — discard, return to `modeNormal`

### Toggle Done
- [ ] `d` or `Space` — toggle `Done` on selected item, call `store.SaveTodos()`, save

### Edit
- [ ] `Enter` in normal mode — populate textinput with current todo text, switch to `modeInput`
- [ ] Saving an edit updates the existing todo's `Text` in place (does not create a new one)

### Delete
- [ ] `x` or `Delete` — remove selected todo from slice, call `store.SaveTodos()`, adjust selection index
- [ ] `D` — remove all todos where `Done == true`, call `store.SaveTodos()`

## Acceptance

- Full list renders correctly with open/done distinction
- All keybindings work and persist to disk
- Adding, editing, completing, and deleting todos all survive a restart

## Notes

<!-- Claude Code: add implementation notes here when done -->
