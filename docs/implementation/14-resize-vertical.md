# 14 — Resize Handling (Vertical Layout)

**State:** `todo`
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

- [ ] Verify `tea.WindowSizeMsg` is still handled in `app.Update()` (from ticket 05) — it should be
- [ ] On `tea.WindowSizeMsg`, update `app.width` and `app.height`
- [ ] Derive `contentHeight = height - 3` and `contentWidth = width`
- [ ] Propagate new dimensions to both `todos.Model` and `notes.Model` on every resize, regardless of which tab is active (so switching tabs after resize is instant)
- [ ] If editor overlay is active on resize, update the `textarea` dimensions to `contentWidth` × `contentHeight`
- [ ] If help overlay is active on resize, reflow the help content to the new width
- [ ] Test with rapid resize (drag terminal corner) — no panics, no visual artifacts

## Acceptance

- Resizing reflows content correctly at any width ≥ 40 cols
- No content clipped or wrapped incorrectly after resize
- Editor and help overlays fill the updated terminal dimensions
- No panics on aggressive or rapid resize

## Notes

<!-- Claude Code: add implementation notes here when done -->
