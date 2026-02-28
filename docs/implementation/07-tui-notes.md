# 07 — Notes Pane

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `05-tui-root.md`
**Blocks:** `08-tui-editor.md`, `11-resize.md`

---

## Goal

Implement the notes pane in `internal/tui/notes.go`. This handles rendering the notes list and triggering the editor for new/edit actions.

## Tasks

### Rendering
- [ ] List notes newest-first (sorted by timestamp filename)
- [ ] Each note shows:
  - Formatted timestamp as header (e.g., `Feb 28 14:30`) or first line of body if it starts with `# `
  - First 2 lines of body text, truncated with `…` if longer
- [ ] Selected note is expanded to show full body content
- [ ] Scroll list if it overflows pane height

### Navigation
- [ ] `j` / `↓` — move selection down
- [ ] `k` / `↑` — move selection up

### New Note
- [ ] `n` — signal root model to switch to `modeEditor` with an empty note

### Edit Note
- [ ] `e` or `Enter` — signal root model to switch to `modeEditor` with selected note's body pre-filled

### Delete Note
- [ ] `x` or `Delete` — show inline confirmation prompt (`y` to confirm, any other key cancels)
- [ ] On confirmation, call `store.DeleteNote(name)`, remove from slice, adjust selection

## Acceptance

- Notes render correctly with timestamp and preview
- Selected note expands to full body
- Navigation works
- Delete requires confirmation and persists

## Notes

<!-- Claude Code: add implementation notes here when done -->
