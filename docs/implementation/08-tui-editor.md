# 08 — Note Editor Overlay

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `07-tui-notes.md`
**Blocks:** `10-tui-help.md` (overlay pattern reference)

---

## Goal

Implement the full-screen note editor in `internal/tui/editor.go`. This is a modal overlay that covers the entire terminal when active.

## Tasks

- [ ] Use `charmbracelet/bubbles/textarea` as the editor widget
- [ ] Editor overlay is rendered by the root model when `mode == modeEditor`
- [ ] The overlay fills the full terminal width and height, covering both panes
- [ ] Show a header bar: `New Note` or `Editing — Feb 28 14:30` depending on context
- [ ] Show a footer hint: `Ctrl+S to save · Esc to discard`

### Opening the Editor
- [ ] Re-enable `a.mode = modeEditor` in `app.go` `Update()` for `enterEditorMsg` (disabled in ticket 07 to prevent dead mode)
- [ ] Root model passes either an empty string (new note) or existing note body (edit)
- [ ] Pre-populate `textarea` with the provided content
- [ ] Focus the textarea immediately on open

### Saving
- [ ] `Ctrl+S` or `Ctrl+D`:
  - If new note: generate timestamp name, call `store.SaveNote(name, body)`, prepend to notes list
  - If editing: call `store.SaveNote(name, body)` with existing name
  - Switch root mode back to `modeNormal`

### Discarding
- [ ] `Esc` with no changes → return to `modeNormal` immediately
- [ ] `Esc` with unsaved changes → show inline prompt: `Discard changes? y/n`
  - `y` → discard and return to `modeNormal`
  - `n` / any other key → stay in editor

### Resize
- [ ] On `tea.WindowSizeMsg`, resize the textarea to fill the updated terminal dimensions

## Acceptance

- Editor opens pre-filled when editing an existing note
- Editor opens empty for new notes
- `Ctrl+S` saves and returns to normal view
- Unsaved `Esc` prompts for confirmation
- Saved note appears immediately in the notes pane

## Notes

<!-- Claude Code: add implementation notes here when done -->
