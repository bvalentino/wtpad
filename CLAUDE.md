# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

wtpad is a terminal-native scratch pad for git worktrees. It provides a tabbed TUI (todos + notes) scoped to the current worktree directory, with data stored in `.wtpad/` as plain markdown files.

## Build & Test

```bash
go build -o wtpad .        # build binary
go build ./...             # verify all packages compile
go test ./...              # run all tests
go test ./internal/store/  # run tests for a single package
```

Dependencies: Go 1.25+, Bubble Tea, Lipgloss, Bubbles (charmbracelet stack).

## Architecture

```
main.go                         # CLI arg routing (add/ls/note/done) or Bubble Tea TUI
internal/model/model.go         # Todo and Note structs (pure data, no logic)
internal/store/store.go         # Disk I/O for .wtpad/ directory (todos + notes)
internal/store/template_store.go # Shared template store at ~/.wtpad/templates/
internal/git/git.go             # Branch detection by reading .git/HEAD directly
internal/tui/
  app.go                        # Root Bubble Tea model, composes panes + overlays
  todos.go                      # Todo tab: list with add/edit/toggle/reorder
  notes.go                      # Notes tab: list + preview
  editor.go                     # Full-screen note editor (modal overlay)
  viewer.go                     # Read-only note viewer (modal overlay)
  templates.go                  # Template import/save modal
  help.go                       # Help overlay (modal)
  styles.go                     # All lipgloss style definitions
```

**Layer rule:** TUI never writes to disk directly — always goes through `store`. Each layer only depends on layers below it.

## Key Design Decisions

- **Bubble Tea composition:** Each pane is its own model with `Update`/`View`. Root `app.go` composes them and forwards messages to the focused pane or active overlay.
- **Tab model:** Root tracks `tabTodos`/`tabNotes`. Only the active tab processes keypresses. Switched via `tab` key.
- **Application modes:** `modeNormal`, `modeInput`, `modeEditor`, `modeViewer`, `modeHelp`, `modeTemplate` — transitions managed by typed messages (`enterEditorMsg`, `exitEditorMsg`, etc.) handled in `app.Update()`.
- **Overlays (editor, viewer, help, template):** Rendered by root when active, covering the full terminal. They receive messages directly from root's `Update`.
- **File-based storage:** Todos are a GFM task list (`- [ ]`/`- [~]`/`- [x]`) in `todos.md`. Notes are individual timestamped `.md` files (`YYYYMMDD-HHMMSS.md`). No JSON.
- **Three todo statuses:** `StatusOpen`, `StatusInProgress` (`- [~]`), `StatusDone`. Not just open/done.
- **Atomic writes:** Write to `.tmp` then `os.Rename` to prevent corruption.
- **No git CLI dependency:** Branch detected by reading `.git/HEAD` directly, with linked worktree support.
- **Synchronous store:** File I/O is fast enough for small markdown files — no `tea.Cmd` wrapping needed.
- **Templates:** Shared across worktrees at `~/.wtpad/templates/`. Same GFM task list format as todos.

## Data Directory

All data lives in `.wtpad/` in the working directory. On first write, `.wtpad/` is auto-added to `.git/info/exclude` (never committed). `todos.md` is the only reserved filename; all other `*.md` files are notes.
