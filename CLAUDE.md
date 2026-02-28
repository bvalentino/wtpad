# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

wtpad is a terminal-native scratch pad for git worktrees. It provides a two-pane TUI (todos + notes) scoped to the current worktree directory, with data stored in `.wtpad/` as plain markdown files.

**Status:** Pre-implementation. The codebase currently contains only documentation. Implementation tickets are in `docs/implementation/` (01 through 12), each with a `State: todo/done` field. Pick the next `todo` ticket whose dependencies are all `done`.

## Build & Test

```bash
go build -o wtpad .        # build binary
go build ./...             # verify all packages compile
go test ./...              # run all tests
go test ./internal/store/  # run tests for a single package
```

Dependencies: Go 1.21+, Bubble Tea, Lipgloss, Bubbles (charmbracelet stack).

## Architecture

```
main.go                     # CLI arg routing → subcommand or Bubble Tea TUI
internal/model/model.go     # Todo and Note structs (pure data, no logic)
internal/store/store.go     # All disk I/O: reads/writes .wtpad/ directory
internal/tui/
  app.go                    # Root Bubble Tea model, composes panes + overlays
  todos.go                  # Left pane: todo list
  notes.go                  # Right pane: note list + preview
  editor.go                 # Full-screen note editor (modal overlay)
  help.go                   # Help overlay (modal)
  statusbar.go              # Bottom status bar
  styles.go                 # All lipgloss style definitions
```

**Layer rule:** TUI never writes to disk directly — always goes through `store`. Each layer only depends on layers below it.

## Key Design Decisions

- **Bubble Tea composition:** Each pane is its own `Model` with `Update`/`View`. Root `app.go` composes them with `lipgloss.JoinHorizontal` and forwards messages to the focused pane.
- **Focus model:** Root tracks `focusTodos`/`focusNotes`. Only the focused pane processes keypresses; the other renders dimmed.
- **Modals (editor, help):** Rendered by root when active, covering the full terminal. Not sub-models — they receive messages directly from root `Update`.
- **Application modes:** `modeNormal`, `modeInput`, `modeEditor`, `modeHelp` — transitions managed by `app.Update()`.
- **File-based storage:** Todos are a GFM task list (`- [ ]`/`- [x]`) in `todos.md`. Notes are individual timestamped `.md` files. No JSON.
- **Atomic writes:** Write to `.tmp` then `os.Rename` to prevent corruption.
- **No git CLI dependency:** Branch detected by reading `.git/HEAD` directly. Worktree name is `filepath.Base(cwd)`.
- **Synchronous store:** File I/O is fast enough for small markdown files — no `tea.Cmd` wrapping needed.

## Data Directory

All data lives in `.wtpad/` in the working directory. On first run, `.wtpad/` is added to `.git/info/exclude` (never committed). `todos.md` is the only reserved filename; all other `*.md` files are notes with timestamp filenames (`YYYYMMDD-HHMMSS.md`).
