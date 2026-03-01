# 14 — Resize Handling (Vertical Layout)

**State:** `done`
> When complete: set State to `done`, fill in the Notes section below.

**Depends on:** `13-layout-redesign.md`
**Blocks:** nothing

---

## Goal

Ensure the vertical layout reflows correctly on terminal resize. This is simpler than the old two-pane resize because there is only one dimension to manage: the content area height.

## How Sizing Works in the New Layout

```
Total height
  - headerHeight  (6 if height >= 30, else 1)
  - 3             (tab strip: 3 lines)
  - 1             (footer)
  = contentHeight

Total width
  = contentWidth  (passed to active tab as-is, no splitting)
```

## Tasks

- [x] Verify `tea.WindowSizeMsg` is still handled in `app.Update()` (from ticket 05) — it should be
- [x] On `tea.WindowSizeMsg`, update `app.width` and `app.height`
- [x] Derive `contentHeight = height - 3` and `contentWidth = width`
- [x] Propagate new dimensions to both `todos.Model` and `notes.Model` on every resize, regardless of which tab is active (so switching tabs after resize is instant)
- [x] If editor overlay is active on resize, update the `textarea` dimensions to `contentWidth` × `contentHeight`
- [x] If help overlay is active on resize, reflow the help content to the new width
- [x] Test with rapid resize (drag terminal corner) — no panics, no visual artifacts

## Acceptance

- Resizing reflows content correctly at any width ≥ 40 cols
- No content clipped or wrapped incorrectly after resize
- Editor and help overlays fill the updated terminal dimensions
- No panics on aggressive or rapid resize

## Notes

All resize handling was already implemented as part of ticket 13 (vertical layout redesign):

- `app.Update()` handles `tea.WindowSizeMsg` at app.go:119-132
- `layoutVertical()` derives contentHeight/contentWidth and propagates to both panes via `SetSize()`
- Editor overlay receives `WindowSizeMsg` directly and updates textarea dimensions
- Help overlay gets width/height updated and uses them for centering
- `contentHeight` is clamped to minimum 1 to prevent panics on very small terminals
- Added 7 resize-specific unit tests covering: both-pane propagation, small terminal, header toggle, rapid resize, editor resize, help resize, and dimension math
