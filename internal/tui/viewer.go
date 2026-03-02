package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// noteTimeFmt is the timestamp format used for note filenames.
const noteTimeFmt = "20060102-150405"

// noteDisplayName formats a note filename into a human-readable string.
func noteDisplayName(name string) string {
	if name == "" {
		return ""
	}
	if t, err := time.Parse(noteTimeFmt, name); err == nil {
		return t.Format("Jan 02 15:04")
	}
	return name
}

// enterViewerMsg signals root to switch to modeViewer.
type enterViewerMsg struct {
	name string
	body string
}

// exitViewerMsg signals root to leave viewer mode.
type exitViewerMsg struct{}

type viewerModel struct {
	name         string   // note filename (for switching to editor)
	rawBody      string   // original markdown (for re-rendering on resize and editor handoff)
	lines        []string // body split into lines
	scrollOffset int
	width        int
	height       int
}

// openViewer prepares the viewer for displaying a note.
func (v viewerModel) openViewer(name, body string, w, h int) viewerModel {
	v.name = name
	v.rawBody = body
	v.scrollOffset = 0
	v.width = w
	v.height = h
	v.lines = strings.Split(body, "\n")
	return v
}

// resize updates dimensions.
func (v viewerModel) resize(w, h int) viewerModel {
	v.width = w
	v.height = h
	v = v.clampScroll()
	return v
}

func (v viewerModel) Update(msg tea.Msg) (viewerModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return v, nil
	}

	contentH := overlayContentHeight(v.height)

	switch keyMsg.String() {
	case "esc", "q":
		return v, func() tea.Msg { return exitViewerMsg{} }
	case "?":
		return v, func() tea.Msg { return enterHelpMsg{} }
	case "e":
		return v, func() tea.Msg {
			return enterEditorMsg{name: v.name, body: v.rawBody}
		}
	case "up", "k":
		if v.scrollOffset > 0 {
			v.scrollOffset--
		}
	case "down", "j":
		v.scrollOffset++
		v = v.clampScroll()
	case "pgup":
		v.scrollOffset -= contentH
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	case "pgdown":
		v.scrollOffset += contentH
		v = v.clampScroll()
	}

	return v, nil
}

func (v viewerModel) View() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}

	// Slice the lines for the visible window
	visible := v.lines
	if v.scrollOffset < len(visible) {
		visible = visible[v.scrollOffset:]
	} else {
		visible = nil
	}

	return renderOverlayBox(noteDisplayName(v.name), visible, v.width, v.height, v.FooterHint())
}

// FooterHint returns the contextual hint for the viewer overlay.
func (v viewerModel) FooterHint() string {
	return "Edit (e) · Back (esc)"
}

func (v viewerModel) clampScroll() viewerModel {
	maxOffset := len(v.lines) - overlayContentHeight(v.height)
	if maxOffset < 0 {
		maxOffset = 0
	}
	if v.scrollOffset > maxOffset {
		v.scrollOffset = maxOffset
	}
	return v
}
