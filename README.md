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
go install github.com/yourname/wtpad@latest
```

Or build from source:

```bash
git clone https://github.com/yourname/wtpad
cd wtpad
go build -o wtpad .
```

## Usage

```bash
wtpad                  # open TUI
wtpad add <text>       # add a todo
wtpad ls               # list todos
wtpad done <n>         # mark todo #n done
wtpad note <text>      # append a note
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

## Data

Everything is stored as plain markdown in `.wtpad/` in your current directory:

```
.wtpad/
├── todos.md            # GFM task list (- [ ] / - [x])
└── 20260228-143022.md  # One file per note (timestamp filename)
```

Files are human-readable and editable with any text editor. On first run, `wtpad` adds `.wtpad/` to `.git/info/exclude` so it stays local to the worktree and is never committed.

## Requirements

- Go 1.21+
- macOS or Linux
