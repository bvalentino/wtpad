# 09 — Status Bar

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `05-tui-root.md`, `12-git-integration.md`
**Blocks:** nothing

---

## Goal

Implement the status bar in `internal/tui/statusbar.go`. It spans the full terminal width and shows context about the current worktree and session state.

## Tasks

- [ ] Define a `StatusBar` struct holding: `dir string`, `branch string`, `openCount int`, `doneCount int`, `hint string`
- [ ] Implement `View(width int) string`:
  - Left section: `<dirname>` + `· <branch>` if branch is non-empty
  - Center section: `<N> open · <N> done`
  - Right section: `hint` string (e.g. `Press ? for help`, or current mode name)
  - Pad sections with spaces so the bar spans the full `width`
- [ ] Style with a distinct background color using lipgloss (e.g. inverted or subtle tint)
- [ ] Root model updates counts from `data` after every save
- [ ] Root model updates `hint` based on current `appMode`

## Acceptance

- Status bar renders at the bottom of the terminal
- Counts stay accurate after add/complete/delete operations
- Bar fills the full terminal width without wrapping

## Notes

<!-- Claude Code: add implementation notes here when done -->
