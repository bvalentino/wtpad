# Changelog

All notable changes to wtpad will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- **`wtpad ai start`**: transitions an existing open task to in-progress instead of always creating a new one
- **`wtpad ai install claude-code`**: installs UserPromptSubmit and Stop hooks for better task tracking reminders

## [0.1.5] - 2026-03-06

### Added

- Direct tab switching with `t`/`n`/`p` keys; `Ctrl+X` to delete all in notes and prompts

### Changed

- Revised keybindings for consistency: `e` to edit, `Delete` to delete, `Ctrl+S` to save template, `Ctrl+T` to set title
- Removed `q` to quit — use `Ctrl+C` only

### Added

- **AI tab**: read-only tab showing `.wtpad/ai.md` with live file watching
- **`wtpad ai` CLI**: `add`, `start`, `done`, `ls`, `clear` for AI agent task tracking
- **`wtpad ai install claude-code`**: set up Claude Code integration with cleanup-aware hooks
- Help overlay (`?`) now supports scrolling with `↑/↓/j/k/PgUp/PgDn` for small terminals
- **Terminal tab title**: set the terminal tab/window title (OSC 2) to match the wtpad title, with "wtpad" as fallback

### Changed

- Centered empty states for all tabs with description and hint text
- Title length limit removed — titles can now be any length
- Title box line width increased from 26 to 35 characters; long titles truncate with ellipsis on the 3rd line

## [0.1.4] - 2026-03-04

### Added

- **Prompts tab**: third tab for managing reusable prompts stored in `~/.wtpad/prompts/`
- Copy prompt to clipboard (`c`) with async write and size cap
- Shared list pane for notes and prompts (scroll, cursor, keyboard navigation)
- **Title**: press `t` in the TUI to set a title displayed as a box overlay on the centered ASCII logo
  - Long titles word-wrap into up to 3 lines inside the box
  - Compact header (short terminals) shows the title right-aligned
  - CLI: `wtpad title <text>`, `wtpad title --clear`, `wtpad title` (show current)

### Fixed

- Clipboard copy (`c`) on prompts now strips the `# Title` heading line

### Changed

- Tab strip renders with 1-space gaps between tabs for cleaner visual separation
- Tightened file permissions to `0o700`/`0o600` across all stores

## [0.1.3] - 2026-03-02

### Fixed

- Wrap long lines in note viewer to stay within overlay borders
- Place cursor at end of text when editing a todo

## [0.1.2] - 2026-03-02

### Changed

- Lower Go directive to 1.24.2 for broader compatibility

## [0.1.1] - 2026-03-02

### Added

- MIT license
- Updated CLAUDE.md to reflect current codebase state

## [0.1.0] - 2026-03-02

Initial public release.

### Added

- Tabbed TUI with Todos and Notes panes (switch with `Tab`)
- **Todos**: add, edit, toggle done/in-progress, reorder (Shift+J/K), copy to clipboard, delete with confirmation
- **Notes**: create, edit, view, delete markdown notes with timestamped filenames
- Full-screen note editor with save/discard and dirty detection
- Read-only note viewer overlay with scrolling
- Help overlay with keybinding reference (`?`)
- Show/hide completed todos (`v`)
- Template system: import and save-as shared templates (`T`/`S`)
- CLI subcommands: `add`, `ls`, `note`, `done` for scriptable access
- Git branch display by reading `.git/HEAD` directly (no git CLI dependency)
- Data stored in `.wtpad/` as plain markdown, auto-excluded from git
- Light mode terminal support
- Go 1.24.2+ compatibility
