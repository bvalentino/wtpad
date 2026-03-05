# wtpad

A terminal-native scratch pad for git worktrees. Manage todos and jot notes without leaving your terminal — scoped to the current worktree, invisible to git.

## Why

When working across multiple worktrees simultaneously (tmux, cmux), context switches are disorienting. `wtpad` gives each worktree its own persistent scratch pad so you always know where you left off.

## Features

- Two-pane TUI: todos on the left, notes on the right
- Vim-friendly keybindings (`j`/`k`, `a`, `d`, `x`, `n`, `e`)
- Data stored in `.wtpad/` in the current directory — auto-ignored by git
- CLI shortcuts for quick actions without opening the TUI
- Single binary, no runtime dependencies

## Install

```bash
go install github.com/bvalentino/wtpad@latest
```

This places the binary in `$GOPATH/bin` (usually `~/go/bin`), which should be on your `PATH`. To update later, run the same command again.

Or build from source:

```bash
git clone https://github.com/bvalentino/wtpad
cd wtpad
go build -o wtpad .
```

To make it available everywhere, symlink the binary to a directory on your `PATH`:

```bash
ln -s "$(pwd)/wtpad" /usr/local/bin/wtpad
```

## Usage

```bash
wtpad                  # open TUI
wtpad add <text>       # add a todo
wtpad ls               # list todos
wtpad done <n>         # mark todo #n done
wtpad note <text>      # append a note
wtpad ai <command>     # AI task tracking (see below)
```

## Keybindings

| Key | Action |
|---|---|
| `Tab` | Switch pane focus |
| `j` / `k` | Navigate |
| `a` | Add todo |
| `d` / `Space` | Toggle done |
| `n` | New note |
| `e` / `Enter` | Edit selected |
| `x` | Delete selected |
| `?` | Help |
| `q` | Quit |

## AI Integration

wtpad can display an AI-managed task list in a dedicated tab, updated in real time via file watching. LLM coding tools (like Claude Code) use the `wtpad ai` CLI to track their work — visible in the TUI as it happens.

### Commands

```bash
wtpad ai start <text>            # add an in-progress task
wtpad ai done <text>             # mark a task as done
wtpad ai add <text>              # add a task for later
wtpad ai ls                      # list current tasks
wtpad ai clear                   # remove all tasks
wtpad ai install claude-code     # set up Claude Code integration
```

### Claude Code Setup

```bash
wtpad ai install claude-code
```

This writes to:

- `.claude/local.md` — instructions telling Claude to use `wtpad ai` commands
- `.claude/settings.local.json` — a `SessionStart` hook that loads current tasks into context

Once installed, Claude Code will automatically track its work in the AI tab.

## Data

Everything is stored as plain markdown in `.wtpad/` in your current directory:

```
.wtpad/
├── todos.md            # GFM task list (- [ ] / - [x])
└── 20260228-143022.md  # One file per note (timestamp filename)
```

Files are human-readable and editable with any text editor. On first run, `wtpad` adds `.wtpad/` to `.git/info/exclude` so it stays local to the worktree and is never committed.

## Development

### Requirements

- Go 1.21+
- macOS or Linux

### Build & test

```bash
go build -o wtpad .   # build binary
go test ./...         # run all tests
```

### Releasing

`go install github.com/bvalentino/wtpad@latest` resolves the latest **tagged** release. After merging changes to `main`, tag a new version for them to be installable:

```bash
git tag v0.x.x
git push origin v0.x.x
```
