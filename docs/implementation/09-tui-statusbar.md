# 09 — Status Bar

**State:** `done`

**Depends on:** `05-tui-root.md`, `12-git-integration.md`
**Blocks:** nothing

---

## Goal

Implement the status bar in `internal/tui/statusbar.go`. It spans the full terminal width and shows context about the current worktree and session state.

## Tasks

- [x] Define a `StatusBar` struct holding: `dir string`, `branch string`, `openCount int`, `doneCount int`, `hint string`
- [x] Implement `View(width int) string`:
  - Left section: `<dirname>` + `· <branch>` if branch is non-empty
  - Center section: `<N> open · <N> done`
  - Right section: `hint` string (e.g. `Press ? for help`, or current mode name)
  - Pad sections with spaces so the bar spans the full `width`
- [x] Style with a distinct background color using lipgloss (e.g. inverted or subtle tint)
- [x] Root model updates counts from `data` after every save
- [x] Root model updates `hint` based on current `appMode`

## Acceptance

- Status bar renders at the bottom of the terminal
- Counts stay accurate after add/complete/delete operations
- Bar fills the full terminal width without wrapping

## Notes

- `statusBarModel` is a plain struct (not a full Bubble Tea model) — it has no `Update` or `Init`. The parent `App` sets its fields via `refreshStatusBar()` and calls `View(width)` at render time.
- Counts derived from `todosModel.Counts()` — no duplicate state between App and statusbar.
- `refreshStatusBar()` called after every pane delegation and mode transition in `App.Update()`.
- Pane height reduced by `statusBarHeight` (1) to make room. `layoutPanes()` and `View()` both use the same calculation.
- Hints per mode: normal → "? help · tab switch", input → "enter confirm · esc cancel", editor → "ctrl+s save · esc discard", help → "esc close".
- Style: `Background(Color("236"))` with `Foreground(Color("252"))` — subtle dark tint matching the selected-item highlight color.
