# 12 — Git Integration

**State:** `done`

**Depends on:** `03-store.md`
**Blocks:** `09-tui-statusbar.md`

---

## Goal

Implement two pieces of git-aware behaviour: reading the current branch name for the status bar, and auto-adding `.wtpad/` to `.git/info/exclude` on first save.

No git CLI dependency. All logic reads the filesystem directly.

## Tasks

### Branch Detection

- [x] Implement `DetectBranch(cwd string) string` in `internal/git/git.go`:
  - Walk up from `cwd` looking for a `.git/` directory or `.git` file (the latter for linked worktrees)
  - If `.git` is a **file** (linked worktree): read it to get the `gitdir:` path, then read `HEAD` from that path
  - If `.git` is a **directory**: read `.git/HEAD` directly
  - Parse `HEAD`:
    - `ref: refs/heads/<branch>` → return `<branch>`
    - Detached HEAD (raw SHA) → return first 7 chars of SHA
  - If `.git` not found anywhere → return `""`
- [x] Call `DetectBranch` once at startup in `main.go` and pass result into the TUI

### Auto-Ignore

- [x] In the store's `ensureDir()` helper (called on first write), check for `.git/info/exclude`:
  - Walk up from `basePath` to find `.git/`
  - If `.git/info/exclude` exists and does not already contain `.wtpad/`, append the line
  - Silently skip if not found (not a git repo)
- [x] This should only run when the `.wtpad/` directory is first created (check if it existed before `MkdirAll`)

## Acceptance

- Status bar shows correct branch name in a standard worktree
- Status bar shows correct branch name in a linked worktree (`git worktree add`)
- Status bar shows short SHA in detached HEAD state
- `.git/info/exclude` contains `.wtpad/` after first save
- No error or panic in non-git directories

## Notes

- Branch detection implemented in a new `internal/git` package rather than in `store.go`, keeping git filesystem logic separate from data I/O.
- `DetectBranch` uses `os.Lstat` to distinguish `.git` files (linked worktrees) from `.git` directories (standard repos).
- Auto-ignore walks up from `.wtpad/` parent to find `.git/info/exclude`. Creates the `info/` directory if needed. Idempotent — checks for existing `.wtpad/` entry before appending.
- Tests cover: standard repo, feature branches, detached HEAD, linked worktrees, subdirectories, non-git dirs, auto-exclude idempotency.
