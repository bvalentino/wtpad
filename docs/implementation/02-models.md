# 02 — Data Models

**State:** `done`

**Depends on:** `01-scaffold.md`
**Blocks:** `03-store.md`, all TUI tickets

---

## Goal

Define the `Todo` and `Note` structs in `internal/model/model.go`. This is the shared data contract used by every other package.

## Tasks

- [x] Define `Todo` struct:
  ```go
  type Todo struct {
      Text string
      Done bool
  }
  ```
  Todos are parsed from / serialized to GFM task list lines (`- [ ] Text` / `- [x] Text`). No IDs needed — position in the file is the identity.
- [x] Define `Note` struct:
  ```go
  type Note struct {
      Name      string    // filename without .md extension (e.g., "20260228-143022")
      Body      string    // full markdown content
      CreatedAt time.Time // parsed from the filename timestamp
  }
  ```
  Each note maps to a single `.md` file in `.wtpad/`.
- [x] Remove the `Data` root container — there is no single JSON blob anymore. The store returns `[]Todo` and `[]Note` separately.
- [x] Confirm `go build ./...` still passes

## Acceptance

Other packages can import `internal/model` and use `Todo` and `Note` without errors.

## Notes

- Scaffold had only a bare `package model` declaration — no `Data` container existed to remove
- Both structs defined with fields matching the spec exactly
- `go build ./...` passes cleanly
