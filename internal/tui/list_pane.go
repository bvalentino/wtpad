package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bvalentino/wtpad/internal/model"
)

// enterEditorMsg signals root to switch to modeEditor.
// Name is empty for a new item, non-empty for editing an existing one.
type enterEditorMsg struct {
	name string
	body string
}

// enterViewerMsg signals root to switch to modeViewer.
type enterViewerMsg struct {
	name string
	body string
}

// listItem is the common representation of a note or prompt within a list pane.
type listItem struct {
	Name      string
	Body      string
	CreatedAt time.Time
}

// listPane holds the shared state and logic for scrollable list panes
// (notes and prompts). Both notesModel and promptsModel embed this struct.
type listPane struct {
	items         []listItem
	cursor        int
	scrollOffset  int
	width         int
	height        int
	focused       bool
	confirmDelete bool
}

func (lp listPane) moveCursor(delta int) listPane {
	lp.cursor += delta
	lp = lp.clampCursor()
	lp = lp.adjustScroll()
	return lp
}

func (lp listPane) clampCursor() listPane {
	if lp.cursor < 0 {
		lp.cursor = 0
	}
	if max := len(lp.items) - 1; lp.cursor > max {
		if max < 0 {
			lp.cursor = 0
		} else {
			lp.cursor = max
		}
	}
	return lp
}

func (lp listPane) availableLines() int {
	h := lp.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

// itemHeight returns the number of terminal lines an item occupies.
// 1 for header only (no body), 2 for header + 1 body line.
func (lp listPane) itemHeight(idx int) int {
	_, body, _ := splitHeadingAndBody(lp.items[idx].Body)
	if body == "" {
		return 1
	}
	return 2
}

func (lp listPane) adjustScroll() listPane {
	if lp.height < 1 || len(lp.items) == 0 {
		return lp
	}
	if lp.cursor < lp.scrollOffset {
		lp.scrollOffset = lp.cursor
	}
	avail := lp.availableLines()
	for {
		used := 0
		cursorVisible := false
		for i := lp.scrollOffset; i < len(lp.items); i++ {
			h := lp.itemHeight(i)
			if i > lp.scrollOffset {
				h++ // blank spacer line between items
			}
			if used+h > avail && i > lp.scrollOffset {
				break
			}
			used += h
			if i == lp.cursor {
				cursorVisible = true
				break
			}
		}
		if cursorVisible {
			break
		}
		lp.scrollOffset++
		if lp.scrollOffset > lp.cursor {
			lp.scrollOffset = lp.cursor
			break
		}
	}

	for lp.scrollOffset > 0 {
		used := 0
		for i := lp.scrollOffset - 1; i < len(lp.items); i++ {
			h := lp.itemHeight(i)
			if i > lp.scrollOffset-1 {
				h++
			}
			used += h
		}
		if used > avail {
			break
		}
		lp.scrollOffset--
	}

	return lp
}

// splitHeadingAndBody extracts a markdown heading from the first line of body.
// If the first line starts with "# ", it returns the heading text and remaining body.
// Otherwise heading is empty and body is returned unchanged.
func splitHeadingAndBody(body string) (heading, rest string, hasHeading bool) {
	if body == "" {
		return "", "", false
	}

	// Find first newline to get the first line without allocating a slice
	firstNL := -1
	for i, ch := range body {
		if ch == '\n' {
			firstNL = i
			break
		}
	}
	var firstLine string
	if firstNL < 0 {
		firstLine = body
	} else {
		firstLine = body[:firstNL]
	}

	if len(firstLine) < 2 || firstLine[0] != '#' || firstLine[1] != ' ' {
		return "", body, false
	}

	heading = firstLine[2:]
	if firstNL >= 0 && firstNL+1 < len(body) {
		rest = body[firstNL+1:]
		// Trim leading newlines from rest
		for len(rest) > 0 && rest[0] == '\n' {
			rest = rest[1:]
		}
	}
	return heading, rest, true
}

// headerText returns the display header for a list item.
// Uses the first line if it starts with "# ", otherwise formats the timestamp.
func headerText(item listItem) string {
	if heading, _, has := splitHeadingAndBody(item.Body); has {
		return heading
	}
	if !item.CreatedAt.IsZero() {
		return item.CreatedAt.Format("Jan 02 15:04")
	}
	return item.Name
}

// previewLines returns the first line of the item body, truncated.
func previewLines(body string, width int) []string {
	_, rest, _ := splitHeadingAndBody(body)
	if rest == "" {
		return nil
	}

	firstLine := strings.SplitN(rest, "\n", 2)[0]
	line := truncate(firstLine, width)
	if strings.Contains(rest, "\n") {
		line = truncate(firstLine, width-1) + "…"
	}
	return []string{line}
}

// renderListLines renders the scrollable list of items as a slice of lines.
func renderListLines(lp listPane) []string {
	visibleLines := lp.availableLines()
	result := make([]string, 0, visibleLines)

	for i := lp.scrollOffset; i < len(lp.items) && len(result) < visibleLines; i++ {
		item := lp.items[i]
		selected := i == lp.cursor && lp.focused

		header := headerText(item)
		lines := previewLines(item.Body, lp.width)

		// Blank line between items for breathing room.
		if len(result) > 0 && len(result) < visibleLines {
			result = append(result, "")
		}

		if len(result) >= visibleLines {
			break
		}

		// Header line
		headerLine := listHeader.Render(header)
		if selected {
			headerLine = listSelected.Render(headerLine)
		}
		result = append(result, headerLine)

		// Body lines
		for _, line := range lines {
			if len(result) >= visibleLines {
				break
			}
			rendered := listPreview.Render(line)
			if selected {
				rendered = listSelected.Render(rendered)
			}
			result = append(result, rendered)
		}
	}

	return result
}

// assembleListView combines the rendered items with padding and a bottom bar
// to produce the final fixed-height view for a list pane.
func assembleListView(lp listPane, barContent string) string {
	visibleLines := lp.availableLines()

	itemLines := renderListLines(lp)
	for len(itemLines) < visibleLines {
		itemLines = append(itemLines, "")
	}
	itemLines = itemLines[:visibleLines]

	itemLines = append(itemLines,
		dividerStyle.Render(strings.Repeat("─", lp.width)),
		barContent,
	)

	return strings.Join(itemLines, "\n")
}

// loadBodies loads bodies for all items using the given load function.
// A nil loadFn is a no-op (used when the store is nil in tests).
func (lp listPane) loadBodies(loadFn func(string) (string, error)) listPane {
	if loadFn == nil {
		return lp
	}
	for i := range lp.items {
		if lp.items[i].Body != "" {
			continue
		}
		body, err := loadFn(lp.items[i].Name)
		if err != nil {
			continue
		}
		lp.items[i].Body = body
	}
	return lp
}

// removeItem removes the item at the given index and adjusts cursor/scroll.
func (lp listPane) removeItem(idx int) listPane {
	lp.items = append(lp.items[:idx], lp.items[idx+1:]...)
	lp = lp.clampCursor()
	lp = lp.adjustScroll()
	return lp
}

// selectedItem returns the item at the cursor, or nil if empty.
func (lp listPane) selectedItem() *listItem {
	if len(lp.items) == 0 {
		return nil
	}
	return &lp.items[lp.cursor]
}

func (lp listPane) setSize(w, h int) listPane {
	lp.width = w
	lp.height = h
	return lp
}

func (lp listPane) setFocus(focused bool) listPane {
	lp.focused = focused
	return lp
}

// handleKey processes shared key events (cursor movement, view, edit, add,
// delete confirmation). Returns the updated pane, an optional command, and
// whether the key was handled.
func (lp listPane) handleKey(keyMsg tea.KeyMsg) (listPane, tea.Cmd, bool) {
	// During confirmDelete, "y" is handled by the consumer (which performs the
	// actual store delete). Any other key cancels.
	if lp.confirmDelete {
		lp.confirmDelete = false
		return lp, nil, true
	}

	switch keyMsg.String() {
	case "down":
		lp = lp.moveCursor(1)
		return lp, nil, true
	case "up":
		lp = lp.moveCursor(-1)
		return lp, nil, true
	case "a":
		return lp, func() tea.Msg { return enterEditorMsg{} }, true
	case "enter":
		if item := lp.selectedItem(); item != nil {
			return lp, func() tea.Msg {
				return enterViewerMsg{name: item.Name, body: item.Body}
			}, true
		}
		return lp, nil, true
	case "e":
		if item := lp.selectedItem(); item != nil {
			return lp, func() tea.Msg {
				return enterEditorMsg{name: item.Name, body: item.Body}
			}, true
		}
		return lp, nil, true
	case "x", "delete":
		if len(lp.items) > 0 {
			lp.confirmDelete = true
		}
		return lp, nil, true
	}

	return lp, nil, false
}

// count returns the number of items in the list.
func (lp listPane) count() int {
	return len(lp.items)
}

// setItems replaces the items, reloads bodies, and adjusts cursor/scroll.
func (lp listPane) setItems(notes []model.Note, loadFn func(string) (string, error)) listPane {
	lp.items = notesToItems(notes)
	lp = lp.loadBodies(loadFn)
	lp = lp.clampCursor()
	lp = lp.adjustScroll()
	return lp
}

func notesToItems(notes []model.Note) []listItem {
	items := make([]listItem, len(notes))
	for i, n := range notes {
		items[i] = listItem{Name: n.Name, Body: n.Body, CreatedAt: n.CreatedAt}
	}
	return items
}
