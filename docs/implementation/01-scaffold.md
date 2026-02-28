# 01 — Project Scaffold

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** nothing
**Blocks:** all other tickets

---

## Goal

Set up the Go module, install all dependencies, and create the empty file structure so every subsequent ticket has a valid place to write code.

## Tasks

- [ ] Run `go mod init github.com/yourname/wtpad`
- [ ] Add dependencies:
  ```bash
  go get github.com/charmbracelet/bubbletea
  go get github.com/charmbracelet/lipgloss
  go get github.com/charmbracelet/bubbles
  ```
- [ ] Create empty files matching the structure in `docs/architecture.md`:
  - `main.go` (package main, empty main func)
  - `internal/model/model.go`
  - `internal/store/store.go`
  - `internal/tui/app.go`
  - `internal/tui/todos.go`
  - `internal/tui/notes.go`
  - `internal/tui/editor.go`
  - `internal/tui/help.go`
  - `internal/tui/statusbar.go`
  - `internal/tui/styles.go`
- [ ] Confirm `go build ./...` passes with no errors

## Acceptance

`go build ./...` succeeds on a clean checkout.

## Notes

<!-- Claude Code: add implementation notes here when done -->
