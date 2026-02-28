# 01 — Project Scaffold

**State:** `done`

**Depends on:** nothing
**Blocks:** all other tickets

---

## Goal

Set up the Go module, install all dependencies, and create the empty file structure so every subsequent ticket has a valid place to write code.

## Tasks

- [x] Run `go mod init github.com/bvalentino/wtpad`
- [x] Add dependencies:
  ```bash
  go get github.com/charmbracelet/bubbletea
  go get github.com/charmbracelet/lipgloss
  go get github.com/charmbracelet/bubbles
  ```
- [x] Create empty files matching the structure in `docs/architecture.md`:
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
- [x] Confirm `go build ./...` passes with no errors

## Acceptance

`go build ./...` succeeds on a clean checkout.

## Notes

- Module initialized as `github.com/bvalentino/wtpad`
- Dependencies installed: bubbletea v1.3.10, lipgloss v1.1.0, bubbles v1.0.0
- Skipped `go mod tidy` to preserve deps in go.mod before they're imported
- All 10 source files created with package declarations only
- `go build ./...` passes cleanly
