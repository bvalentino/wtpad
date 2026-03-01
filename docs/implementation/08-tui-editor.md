# 08 — Note Editor Overlay

**State:** `done`

**Depends on:** `07-tui-notes.md`
**Blocks:** `10-tui-help.md` (overlay pattern reference)

---

## Goal

Implement the full-screen note editor in `internal/tui/editor.go`. This is a modal overlay that covers the entire terminal when active.

## Tasks

- [x]Use `charmbracelet/bubbles/textarea` as the editor widget
- [x]Editor overlay is rendered by the root model when `mode == modeEditor`
- [x]The overlay fills the full terminal width and height, covering both panes
- [x]Show a header bar: `New Note` or `Editing — Feb 28 14:30` depending on context
- [x]Show a footer hint: `Ctrl+S to save · Esc to discard`

### Opening the Editor
- [x]Re-enable `a.mode = modeEditor` in `app.go` `Update()` for `enterEditorMsg` (disabled in ticket 07 to prevent dead mode)
- [x]Root model passes either an empty string (new note) or existing note body (edit)
- [x]Pre-populate `textarea` with the provided content
- [x]Focus the textarea immediately on open

### Saving
- [x]`Ctrl+S` or `Ctrl+D`:
  - If new note: generate timestamp name, call `store.SaveNote(name, body)`, prepend to notes list
  - If editing: call `store.SaveNote(name, body)` with existing name
  - Switch root mode back to `modeNormal`

### Discarding
- [x]`Esc` with no changes → return to `modeNormal` immediately
- [x]`Esc` with unsaved changes → show inline prompt: `Discard changes? y/n`
  - `y` → discard and return to `modeNormal`
  - `n` / any other key → stay in editor

### Resize
- [x]On `tea.WindowSizeMsg`, resize the textarea to fill the updated terminal dimensions

## Acceptance

- Editor opens pre-filled when editing an existing note
- Editor opens empty for new notes
- `Ctrl+S` saves and returns to normal view
- Unsaved `Esc` prompts for confirmation
- Saved note appears immediately in the notes pane

## Notes

- Editor implemented in `internal/tui/editor.go` using `bubbles/textarea`
- Modal overlay rendered by root `App.View()` when `mode == modeEditor`, covering both panes
- Header shows "New Note" or "Editing — timestamp" parsed from note filename
- Footer shows save/discard hints, switches to "Discard changes? y/n" when dirty
- Save errors displayed inline in footer; editor stays open for retry
- `openEditor()` resets all state (confirmDiscard, err) to prevent stale state between sessions
- Editor styles added to `styles.go`: `editorHeader`, `editorFooter`, `editorConfirm`
