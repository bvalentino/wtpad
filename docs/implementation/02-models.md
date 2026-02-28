# 02 — Data Models

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `01-scaffold.md`
**Blocks:** `03-store.md`, all TUI tickets

---

## Goal

Define the `Todo` and `Note` structs in `internal/model/model.go`. This is the shared data contract used by every other package.

## Tasks

- [ ] Define `Todo` struct:
  ```go
  type Todo struct {
      Text string
      Done bool
  }
  ```
  Todos are parsed from / serialized to GFM task list lines (`- [ ] Text` / `- [x] Text`). No IDs needed — position in the file is the identity.
- [ ] Define `Note` struct:
  ```go
  type Note struct {
      Name      string    // filename without .md extension (e.g., "20260228-143022")
      Body      string    // full markdown content
      CreatedAt time.Time // parsed from the filename timestamp
  }
  ```
  Each note maps to a single `.md` file in `.wtpad/`.
- [ ] Remove the `Data` root container — there is no single JSON blob anymore. The store returns `[]Todo` and `[]Note` separately.
- [ ] Confirm `go build ./...` still passes

## Acceptance

Other packages can import `internal/model` and use `Todo` and `Note` without errors.

## Notes

<!-- Claude Code: add implementation notes here when done -->
