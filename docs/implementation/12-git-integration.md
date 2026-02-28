# 12 — Git Integration

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below, and remove this line.

**Depends on:** `03-store.md`
**Blocks:** `09-tui-statusbar.md`

---

## Goal

Implement two pieces of git-aware behaviour: reading the current branch name for the status bar, and auto-adding `.wtpad/` to `.git/info/exclude` on first save.

No git CLI dependency. All logic reads the filesystem directly.

## Tasks

### Branch Detection

- [ ] Implement `DetectBranch(cwd string) string` in `internal/store/store.go` (or a `git.go` helper):
  - Walk up from `cwd` looking for a `.git/` directory or `.git` file (the latter for linked worktrees)
  - If `.git` is a **file** (linked worktree): read it to get the `gitdir:` path, then read `HEAD` from that path
  - If `.git` is a **directory**: read `.git/HEAD` directly
  - Parse `HEAD`:
    - `ref: refs/heads/<branch>` → return `<branch>`
    - Detached HEAD (raw SHA) → return first 7 chars of SHA
  - If `.git` not found anywhere → return `""`
- [ ] Call `DetectBranch` once at startup in `main.go` and pass result into the TUI

### Auto-Ignore

- [ ] In the store's `ensureDir()` helper (called on first write), check for `.git/info/exclude`:
  - Walk up from `basePath` to find `.git/`
  - If `.git/info/exclude` exists and does not already contain `.wtpad/`, append the line
  - Silently skip if not found (not a git repo)
- [ ] This should only run when the `.wtpad/` directory is first created (check if it existed before `MkdirAll`)

## Acceptance

- Status bar shows correct branch name in a standard worktree
- Status bar shows correct branch name in a linked worktree (`git worktree add`)
- Status bar shows short SHA in detached HEAD state
- `.git/info/exclude` contains `.wtpad/` after first save
- No error or panic in non-git directories

## Notes

<!-- Claude Code: add implementation notes here when done -->
