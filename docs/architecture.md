# wtpad — Architecture

High-level map of the codebase. Read this to orient yourself before diving into any implementation ticket.

---

## Directory Structure

```
wtpad/
├── main.go                   # Entrypoint: parses CLI args, routes to TUI or CLI handler
├── go.mod
├── go.sum
├── internal/
│   ├── store/
│   │   └── store.go          # All data persistence: load, save, atomic write
│   ├── model/
│   │   └── model.go          # Todo and Note structs; no business logic
│   └── tui/
│       ├── app.go            # Root Bubble Tea model — composes all sub-components
│       ├── todos.go          # Todo pane: list, selection, inline add/edit
│       ├── notes.go          # Notes pane: list, selection, preview
│       ├── editor.go         # Full-screen note editor overlay
│       ├── help.go           # Help overlay
│       ├── statusbar.go      # Bottom status bar
│       └── styles.go         # All lipgloss style definitions
└── docs/                     # This documentation
```

---

## Layers

```
main.go
  └── CLI args? → run CLI command directly via store
  └── No args?  → start Bubble Tea program

tui/app.go  (root model)
  ├── tui/todos.go      (left pane)
  ├── tui/notes.go      (right pane)
  ├── tui/editor.go     (modal overlay — note editor)
  ├── tui/help.go       (modal overlay — keybinding help)
  └── tui/statusbar.go  (bottom bar)

internal/store/store.go
  └── internal/model/model.go
```

Each layer only depends on layers below it. The TUI never writes to disk directly — it always goes through `store`.

---

## Key Design Decisions

**Bubble Tea composition** — Each pane (`todos.go`, `notes.go`) is its own Bubble Tea `Model` with its own `Update` and `View`. The root `app.go` model holds them as fields, forwards messages to the focused pane, and assembles the final view using `lipgloss.JoinHorizontal`.

**Focus model** — The root model tracks which pane is active (`focusTodos` or `focusNotes`). `Tab` toggles focus. Only the focused pane processes keypresses; the other renders in a dimmed state.

**Modals** — The editor and help overlay are rendered by the root model when active, covering the entire terminal. They are not sub-models; they receive messages directly from the root `Update`.

**Store is synchronous** — File I/O is fast enough for this use case (small JSON file). No `tea.Cmd` wrapping needed for reads/writes. If this ever becomes a bottleneck, wrap in a `tea.Cmd` and handle a result message.

**No git CLI dependency** — Git branch is detected by reading `.git/HEAD` directly. Worktree name is `filepath.Base(cwd)`. No `exec.Command("git", ...)` calls anywhere.

**Atomic writes** — Store writes to `data.json.tmp` then calls `os.Rename`. This is atomic on POSIX systems and prevents corruption if the process is killed mid-write.

---

## Data Flow

```
User keypress
  → tea.KeyMsg
  → app.Update()
    → delegate to focused pane (todos.Update / notes.Update)
    → pane returns updated model + optional store mutation
  → store.Save(data)
  → app.View()
    → todos.View() + notes.View() joined horizontally
    → statusbar.View() appended below
```

---

## Application Modes

The root model tracks the current mode as an enum:

| Mode | Description |
|---|---|
| `modeNormal` | Default — both panes visible, navigation active |
| `modeInput` | Todo inline add/edit input is open |
| `modeEditor` | Full-screen note editor overlay is active |
| `modeHelp` | Help overlay is active |

Mode transitions are the responsibility of `app.Update()`.

---

## Implementation Ticket Index

See `docs/implementation/` for all tickets. Each ticket has a `State` of either `todo` or `done`. Claude Code should pick the next `todo` ticket whose dependencies are all `done`. Recommended implementation order:

1. `01-scaffold.md` — Go module, dependencies, empty main ✓
2. `02-models.md` — Todo and Note structs ✓
3. `03-store.md` — JSON persistence layer ✓
4. `04-cli.md` — CLI subcommands ✓
5. `05-tui-root.md` — Bubble Tea root model and layout shell ✓
6. `06-tui-todos.md` — Todo pane ✓
7. `07-tui-notes.md` — Notes pane ✓
8. `08-tui-editor.md` — Note editor overlay ✓
9. `09-tui-statusbar.md` — Status bar ✓
10. `10-tui-help.md` — Help overlay ✓
11. `11-resize.md` — Terminal resize handling ✓ (superseded, see 13 & 14)
12. `12-git-integration.md` — Branch detection, auto-ignore ✓
13. `13-layout-redesign.md` — Vertical tab layout redesign
14. `14-resize-vertical.md` — Resize handling for vertical layout
