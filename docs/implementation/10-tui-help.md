# 10 — Help Overlay

**State:** `done`

**Depends on:** `05-tui-root.md`, `08-tui-editor.md` (overlay pattern reference)
**Blocks:** nothing

---

## Goal

Implement the help overlay in `internal/tui/help.go`. Pressing `?` from any normal mode shows a full-screen keybinding reference. Any key dismisses it.

## Tasks

- [x] The overlay is rendered by the root model when `mode == modeHelp`
- [x] It covers the full terminal (same pattern as `editor.go`)
- [x] Display a structured keybinding table, grouped by context:
  - Global
  - Todo pane
  - Notes pane
  - Note editor
- [x] Any keypress while in `modeHelp` → switch back to `modeNormal`
- [x] Style the overlay with a clear visual distinction from the main UI (border, background, centered title)

## Content

```
wtpad — keyboard shortcuts

Global
  Tab        Switch pane focus
  ?          Toggle this help
  q / Ctrl+C Quit

Todos
  j / k      Navigate
  a          Add todo
  d / Space  Toggle done
  Enter      Edit selected
  x          Delete selected
  D          Delete all completed

Notes
  j / k      Navigate
  n          New note
  e / Enter  Edit selected
  x          Delete selected

Editor
  Ctrl+S     Save
  Esc        Discard / close
```

## Acceptance

- `?` opens overlay from any normal mode
- All keybindings are listed and accurate
- Any key dismisses the overlay and returns to normal mode

## Notes

- `helpModel` in `help.go` with `Update()` (any key → `exitHelpMsg`) and `View()` (centered keybinding table)
- Follows editor overlay pattern: full-screen takeover in `app.View()`, message-based transitions in `app.Update()`
- Styles in `styles.go`: `helpTitle`, `helpSection`, `helpKey`, `helpDesc` using existing color palette
- `?` triggers from `modeNormal`, any keypress dismisses back to `modeNormal`
- `WindowSizeMsg` propagated to help pane for proper centering on resize
