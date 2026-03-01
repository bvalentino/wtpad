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
wtpad · main
┌──────────┐
│ Todo (t) │ Notes (n)
├──────────────────────┐
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

- **Header:** single line — `wtpad · <branch>` (replaces ASCII art for now, see note below)
- **Tab strip:** two tabs — `Todo (t)` and `Notes (n)`. Active tab has a visible selected style (border bottom or highlight). Inactive tab is dimmed.
- **Content area:** full width, fills remaining height between tab strip and footer
- **Footer:** single line — `<N> open · <N> done · ? help`

> **On the ASCII art header:** it's a nice touch but costs 6 lines of vertical space. Omit it for now and leave a `// TODO: optional ASCII art header` comment in the code. Can be added as a v1.1 feature, possibly only shown when height > threshold.

## Tasks

### Tab Strip
- [ ] Define a `activeTab` enum: `tabTodos | tabNotes`
- [ ] Render tab strip as a single line with two tab labels
- [ ] Active tab: bold, underlined or bordered; inactive tab: dimmed
- [ ] `t` switches to todos tab; `n` switches to notes tab (from anywhere in normal mode)
- [ ] `Tab` key cycles between the two tabs (existing behaviour, just rewired)

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
- [ ] Header: `wtpad · <branch>` — single line, subtle styling
- [ ] Footer replaces current status bar: `<N> open · <N> done · ? help · tab switch`
- [ ] Footer spans full width

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
