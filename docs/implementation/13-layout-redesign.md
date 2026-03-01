# 13 — Vertical Layout Redesign

**State:** `todo`
> When complete: set State to `done`, fill in the Notes section below.

**Depends on:** `06-tui-todos.md`, `07-tui-notes.md`, `08-tui-editor.md`, `09-tui-statusbar.md`
**Blocks:** `14-resize-vertical.md`

---

## Goal

Replace the current side-by-side two-pane layout with a vertical single-column layout optimised for use in a narrow terminal split (e.g. 1/3 of total screen width). Todos and Notes become tabs that share the full width and height of the window.

## New Layout

```
  ___       _________              _________
  __ |     / /__  __/_____________ ______  /
  __ | /| / /__  /  ___  __ \  __ `/  __  /
  __ |/ |/ / _  /   __  /_/ / /_/ // /_/ /
  ____/|__/  /_/    _  .___/\__,_/ \__,_/
                    /_/

┌──────────┐
│ Todo (t) │ Notes (n)
│ ──────── └───────────┐
│                      │
│  ○ Task 1            │
│                      │
│  ○ Task 2            │
│                      │
│  Add (a)             │
│                      │
│  ───────────────     │
│                      │
│  ✓ Completed task    │
│                      │
└──────────────────────┘
3 open · 1 done · ? help
```

- **ASCII art header:** always shown when `height >= 30`; collapses to a single line `wtpad · <branch>` when the terminal is too short
- **Tab strip:** two tabs — `Todo (t)` and `Notes (n)`. Active tab is rendered as a box with an inner bottom rule and `└` connecting to the content area. Inactive tab is a plain dimmed label to the right.
- **Content area:** full width, fills remaining height between tab strip and footer
- **Footer:** single line — `<N> open · <N> done · ? help · tab switch`

### Tab Chrome Detail

The active tab uses this exact three-line structure:

```
┌──────────┐
│ Todo (t) │ Notes (n)
│ ──────── └───────────┐
```

- Line 1: `┌` + `─` × (label width) + `┐`
- Line 2: `│` + ` label ` + `│` + ` ` + inactive label (dimmed)
- Line 3: `│` + ` ` + `─` × (label width - 2) + ` ` + `└` + `─` × (remaining width) + `┐`

The `└` on line 3 is the bottom-right corner of the active tab box, simultaneously acting as the top-left corner of the content area. The content area left border (`│`) is already established by the tab box left side.

When the **Notes tab** is active, the layout mirrors horizontally:

```
 Todo (t) ┌──────────┐
           │ Notes (n)│
┌──────────┘ ──────── │
│                     │
```

## Tasks

### Tab Strip
- [ ] Define a `activeTab` enum: `tabTodos | tabNotes`
- [ ] Render the tab strip as three lines (see Tab Chrome Detail above)
- [ ] Active tab: full box (`┌─┐ │ │ │──└`) with label inside; the `└` on line 3 connects seamlessly to the content area top-right
- [ ] Inactive tab: plain label, dimmed, rendered to the right of the active tab box on line 2
- [ ] `t` switches to todos tab; `n` switches to notes tab (from anywhere in normal mode)
- [ ] `Tab` key cycles between the two tabs
- [ ] When Notes is active, mirror the layout: inactive `Todo (t)` label on the left, active Notes box on the right, `┘` connecting to content area top-left

### Content Area
- [ ] Remove `lipgloss.JoinHorizontal` layout from `app.go`
- [ ] Root `View()` now renders: header + tab strip + active tab content + footer
- [ ] Active tab content fills `height - 3` lines (1 header + 1 tab strip + 1 footer)
- [ ] Pass full terminal `width` to whichever tab is active

### Todo Tab Content
- [ ] Render open todos first, each on its own line with blank line between items for breathing room
- [ ] Render `Add (a)` hint below open todos
- [ ] Render a horizontal divider (`─` repeated to full width) between open and completed sections
- [ ] Render completed todos below divider (dimmed, strikethrough)
- [ ] Remove any previous left-pane width constraints — now full width

### Notes Tab Content
- [ ] Notes render full width (no right-side truncation)
- [ ] Selected note expands to show full body inline
- [ ] Otherwise same behaviour as current notes pane

### Header & Footer
- [ ] If `height >= 30`: render the full ASCII art header (6 lines) above the tab strip
- [ ] If `height < 30`: render a single line `wtpad · <branch>` instead
- [ ] Footer spans full width: `<N> open · <N> done · ? help · tab switch`

### Cleanup
- [ ] Remove `todos.go` and `notes.go` width/height logic tied to the old 40/60 split
- [ ] Remove `statusbar.go` and inline the footer into `app.go` View (it's simple enough)
- [ ] Update `styles.go` with any new styles needed

## Acceptance

- App renders correctly in a narrow terminal (~80 cols or less)
- Switching tabs with `t`, `n`, and `Tab` works
- Todo and Notes content use full available width
- Open/done separator is visible in the todo tab
- Header and footer are single lines each
- No regressions in add, edit, delete, complete for todos or notes

## Notes

<!-- Claude Code: add implementation notes here when done -->
