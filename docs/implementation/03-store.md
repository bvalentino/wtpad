# 03 — Store (Persistence Layer)

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `02-models.md`
**Blocks:** `04-cli.md`, `05-tui-root.md`

---

## Goal

Implement `internal/store/store.go` — the single place responsible for reading and writing files in `.wtpad/`. All other packages call this; none write to disk directly.

## Data Format

**todos.md** — GFM task list:
```md
- [ ] Buy groceries
- [x] Fix login bug
- [ ] Review PR #42
```

**`<YYYYMMDD-HHMMSS>.md`** — One file per note, directly in `.wtpad/`. Filename is a timestamp (e.g., `20260228-143022.md`). Content is freeform markdown. `todos.md` is reserved.

## Tasks

- [ ] Define a `Store` struct holding the resolved base path (`.wtpad/`)
- [ ] Implement `New(dir string) (*Store, error)`:
  - Accepts a directory path (typically `cwd`)
  - Sets `basePath` to `<dir>/.wtpad/`
  - Does **not** create the directory yet (lazy init)
- [ ] **Todos**:
  - [ ] Implement `LoadTodos() ([]model.Todo, error)`:
    - If `todos.md` does not exist, return empty slice (not an error)
    - Parse each `- [ ] Text` / `- [x] Text` line into a `Todo`
  - [ ] Implement `SaveTodos(todos []model.Todo) error`:
    - Ensure `.wtpad/` directory exists (`os.MkdirAll`)
    - Render todos as GFM task list lines
    - Write to `todos.md.tmp` then `os.Rename` (atomic)
- [ ] **Notes**:
  - [ ] Implement `ListNotes() ([]model.Note, error)`:
    - Scan `.wtpad/*.md`, excluding `todos.md`
    - Sort by filename descending (newest first — timestamps sort naturally)
    - Return slice of `Note` structs
  - [ ] Implement `LoadNote(name string) (*model.Note, error)`:
    - Read `.wtpad/<name>.md` and return a `Note`
  - [ ] Implement `SaveNote(name string, body string) error`:
    - Write to `.wtpad/<name>.md.tmp` then `os.Rename` (atomic)
    - If name is empty, generate a new timestamp name (`time.Now().Format("20060102-150405")`)
  - [ ] Implement `DeleteNote(name string) error`:
    - Remove `.wtpad/<name>.md`
- [ ] Implement `Dir() string` returning the `.wtpad/` directory path
- [ ] Write tests: round-trip todos, round-trip notes, handle missing files gracefully

## Acceptance

- `SaveTodos` then `LoadTodos` round-trips data without loss
- `SaveNote` then `LoadNote` round-trips content without loss
- `ListNotes` returns notes sorted newest-first
- `LoadTodos` on a missing file returns empty slice, not an error
- No partial writes visible (tmp + rename pattern)

## Notes

<!-- Claude Code: add implementation notes here when done -->
