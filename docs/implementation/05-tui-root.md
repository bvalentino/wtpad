# 05 — TUI Root Model & Layout Shell

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `03-store.md`, `04-cli.md`
**Blocks:** `06-tui-todos.md`, `07-tui-notes.md`, `09-tui-statusbar.md`, `10-tui-help.md`

---

## Goal

Implement the root Bubble Tea model in `internal/tui/app.go`. This is the backbone of the TUI — it owns the layout, manages focus, and composes all sub-components. At the end of this ticket the app should launch and show an empty two-pane shell.

## Tasks

- [ ] Define the `App` struct:
  ```go
  type App struct {
      store    *store.Store
      todos    []model.Todo
      notes    []model.Note
      width    int
      height   int
      focus    focusPane  // focusTodos | focusNotes
      mode     appMode    // modeNormal | modeInput | modeEditor | modeHelp
      todosPane todos.Model
      notesPane notes.Model
      // editor and help added in later tickets
  }
  ```
- [ ] Define `focusPane` and `appMode` as typed enums with `iota`
- [ ] Implement `New(s *store.Store, todos []model.Todo, notes []model.Note) App`
- [ ] Implement `Init() tea.Cmd` — return nil
- [ ] Implement `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
  - Handle `tea.WindowSizeMsg` → update `width`/`height`, propagate to panes
  - Handle `tea.KeyMsg`:
    - `Tab` → toggle focus
    - `q` / `ctrl+c` → `tea.Quit`
  - Delegate remaining key messages to the focused pane
- [ ] Implement `View() string`:
  - Use `lipgloss.JoinHorizontal` to place todos (40%) and notes (60%) side by side
  - Append status bar below (stubbed as empty string for now)
  - Return the assembled string
- [ ] In `main.go`, wire up: create store → load todos + list notes → `tui.New(store, todos, notes)` → `tea.NewProgram(app, tea.WithAltScreen()).Run()`
- [ ] Confirm the app launches, shows two empty bordered panes, and quits on `q`

## Acceptance

- App launches without panic
- Two panes visible side by side with borders
- `Tab` visually shifts focus (border highlight)
- `q` quits cleanly

## Notes

<!-- Claude Code: add implementation notes here when done -->
