# 04 — CLI Subcommands

**State:** `done`

**Depends on:** `03-store.md`
**Blocks:** `05-tui-root.md` (main.go routing)

---

## Goal

Implement the non-TUI CLI subcommands in `main.go`. These allow quick actions from a shell prompt or tmux without opening the TUI.

## Subcommands

```bash
wtpad add <text>    # add a todo
wtpad ls            # print todos to stdout
wtpad note <text>   # create a new note
wtpad done <n>      # mark todo #n done (1-indexed from ls)
```

## Tasks

- [x] In `main.go`, check `os.Args` before starting the TUI:
  - If no args → fall through to TUI (ticket `05-tui-root.md`)
  - If first arg matches a subcommand → handle and exit
- [x] Implement `cmdAdd(store, args)`:
  - Join remaining args as the todo text
  - Load todos, append new `Todo{Text: text, Done: false}`, save
  - Print confirmation
- [x] Implement `cmdLs(store)`:
  - Load todos, print open todos numbered 1…N
  - Format: `1. Fix auth bug`
  - Completed todos printed after open ones, prefixed with `✓`
- [x] Implement `cmdNote(store, args)`:
  - Join remaining args as the note body
  - Generate a timestamp name, save as `.wtpad/<YYYYMMDD-HHMMSS>.md`
  - Print confirmation with the filename
- [x] Implement `cmdDone(store, args)`:
  - Parse arg as integer N
  - Load todos, find the Nth open todo (same order as `ls`)
  - Set `Done = true`, save
  - Print confirmation or error if N is out of range
- [x] Print usage and exit 1 for unknown subcommands

## Acceptance

- All subcommands work end-to-end against real files in `.wtpad/`
- `add` / `done` / `ls` read and write `todos.md`
- `note` creates a new `.md` file in `.wtpad/`

## Notes

- All subcommands implemented as thin wrappers around `store` package
- `cmdNote` passes empty name to `SaveNote`, letting the store generate the timestamp and handle collisions
- `cmdDone` indexes open todos in file order, matching `cmdLs` numbering
- No-args case prints placeholder message; TUI entry point deferred to ticket 05
- All errors printed to stderr with `os.Exit(1)`
