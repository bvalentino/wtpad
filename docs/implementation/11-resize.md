# 11 — Terminal Resize Handling

**State:** `canceled`

**Depends on:** `06-tui-todos.md`, `07-tui-notes.md`
**Blocks:** nothing

---

## Goal

Ensure the layout reflows correctly when the terminal is resized. This is a cross-cutting concern that touches the root model and each pane.

## Tasks

- [ ] Root model already handles `tea.WindowSizeMsg` from ticket `05` — verify width/height are stored
- [ ] On resize, recalculate pane widths:
  - Todos pane: `int(float64(width) * 0.4)`
  - Notes pane: `width - todosPaneWidth - 1` (subtract divider)
  - Pane height: `height - statusBarHeight` (1 line)
- [ ] Propagate new dimensions to `todos.Model` and `notes.Model`
- [ ] If editor overlay is active, resize the `textarea` to the new terminal dimensions
- [ ] Verify no visual artifacts or panics on rapid resize (drag terminal corner)

## Acceptance

- Resizing the terminal reflows both panes proportionally
- No content is clipped or wrapped incorrectly after resize
- Editor overlay fills updated dimensions
- No panics on aggressive resize

## Notes

Superseded by the vertical layout redesign. The tea.WindowSizeMsg wiring from ticket 05 is already in place. Detailed resize handling for the new layout is covered in 13-layout-redesign.md and 14-resize-vertical.md.
