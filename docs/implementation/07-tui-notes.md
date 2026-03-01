# 07 — Notes Pane

**State:** `done`

**Depends on:** `05-tui-root.md`
**Blocks:** `08-tui-editor.md`, `11-resize.md`

---

## Goal

Implement the notes pane in `internal/tui/notes.go`. This handles rendering the notes list and triggering the editor for new/edit actions.

## Tasks

### Rendering
- [x] List notes newest-first (sorted by timestamp filename)
- [x] Each note shows:
  - Formatted timestamp as header (e.g., `Feb 28 14:30`) or first line of body if it starts with `# `
  - First 2 lines of body text, truncated with `…` if longer
- [x] Selected note is expanded to show full body content
- [x] Scroll list if it overflows pane height

### Navigation
- [x] `j` / `↓` — move selection down
- [x] `k` / `↑` — move selection up

### New Note
- [x] `n` — signal root model to switch to `modeEditor` with an empty note

### Edit Note
- [x] `e` or `Enter` — signal root model to switch to `modeEditor` with selected note's body pre-filled

### Delete Note
- [x] `x` or `Delete` — show inline confirmation prompt (`y` to confirm, any other key cancels)
- [x] On confirmation, call `store.DeleteNote(name)`, remove from slice, adjust selection

## Acceptance

- Notes render correctly with timestamp and preview
- Selected note expands to full body
- Navigation works
- Delete requires confirmation and persists

## Notes

- `notesModel` lazily loads note bodies on cursor move (via `ensureBodyLoaded`)
- `enterEditorMsg` signals root to switch to `modeEditor`; editor pane itself is ticket 08
- Delete uses inline confirmation (`confirmDelete` bool); `y` confirms, any other key cancels
- Note header uses `# ` first line as title if present, otherwise formats timestamp
- Collapsed notes show 2-line preview; selected note expands to full body
- 10 tests in `notes_test.go` covering rendering, navigation, signals, delete, and SetNotes
