package tui

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// clearAIStatusMsg clears the transient status message after a delay.
type clearAIStatusMsg struct{}

func clearAIStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearAIStatusMsg{}
	})
}

// aiFileChangedMsg signals that ai.md was created, modified, or removed on disk.
type aiFileChangedMsg struct{}

// aiLLMPrompt is the text copied to the clipboard when the user presses "p".
const aiLLMPrompt = `# AI Task List — .wtpad/ai.md

Maintain a GFM task list on .wtpad/ai.md in the current working directory.
This file is displayed in a different terminal as a read-only task list, and as a way to follow what's being done.

## Format

- [ ] Open task
- [~] In-progress task
- [x] Completed task

## Workflow

1. Before starting work, add the task to ai.md and mark it in-progress (- [~])
2. Do the work
3. Mark it completed (- [x]) when done

Always update ai.md BEFORE starting a task so the user can see what you're working on in real time.

## Conventions

- One task per line, plain text after the checkbox
- Keep it short — these are displayed in a narrow terminal pane
- Update statuses as work progresses
`

type aiModel struct {
	todos     []model.Todo
	store     *store.Store
	cursor    int
	scrollOff int
	width     int
	height    int
	textWidth int
	focused   bool
	statusMsg string
	confirm   confirmKind
}

func newAI(todos []model.Todo, s *store.Store) aiModel {
	return aiModel{
		todos: todos,
		store: s,
	}
}

func (m aiModel) SetSize(w, h int) aiModel {
	m.width = w
	m.height = h
	m.textWidth = w - todoPrefixWidth
	if m.textWidth < 1 {
		m.textWidth = 1
	}
	m = m.adjustScroll()
	return m
}

func (m aiModel) SetFocus(focused bool) aiModel {
	m.focused = focused
	return m
}

func (m aiModel) Update(msg tea.Msg) (aiModel, tea.Cmd) {
	switch msg.(type) {
	case clearAIStatusMsg:
		m.statusMsg = ""
		return m, nil
	case aiFileChangedMsg:
		m = m.reload()
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// Handle clear confirmation.
	if m.confirm != confirmNone {
		if keyMsg.String() == "y" {
			if err := m.store.ClearAI(); err != nil {
				log.Printf("wtpad: failed to clear ai.md: %v", err)
			} else {
				m.todos = nil
				m.cursor = 0
				m.scrollOff = 0
			}
		}
		m.confirm = confirmNone
		return m, nil
	}

	switch keyMsg.String() {
	case "down":
		m = m.moveCursor(1)
	case "up":
		m = m.moveCursor(-1)
	case "X":
		if len(m.todos) > 0 {
			m.confirm = confirmPurge
		}
	case "c":
		if len(m.todos) > 0 {
			if clipboard.Unsupported {
				m.statusMsg = "Clipboard not available"
				return m, clearAIStatusAfter(2 * time.Second)
			}
			if err := clipboard.WriteAll(m.todos[m.cursor].Text); err != nil {
				m.statusMsg = "Copy failed"
			} else {
				m.statusMsg = "Copied!"
			}
			return m, clearAIStatusAfter(2 * time.Second)
		}
	case "p":
		if clipboard.Unsupported {
			m.statusMsg = "Clipboard not available"
			return m, clearAIStatusAfter(2 * time.Second)
		}
		if err := clipboard.WriteAll(aiLLMPrompt); err != nil {
			m.statusMsg = "Copy failed"
		} else {
			m.statusMsg = "Prompt copied!"
		}
		return m, clearAIStatusAfter(2 * time.Second)
	}

	return m, nil
}

func (m aiModel) View() string {
	if len(m.todos) == 0 {
		line1 := "AI-managed task list."
		line2 := hintStyle.Render("Waiting for ai.md to appear…")

		lines := []string{line1, line2}
		totalLines := len(lines)
		topPad := (m.height - totalLines) / 2
		if topPad < 0 {
			topPad = 0
		}

		var b strings.Builder
		for i := 0; i < topPad; i++ {
			b.WriteString("\n")
		}
		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			leftPad := (m.width - lineWidth) / 2
			if leftPad < 0 {
				leftPad = 0
			}
			b.WriteString(strings.Repeat(" ", leftPad) + line)
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
		}
		return b.String()
	}

	var b strings.Builder
	linesUsed := 0
	visibleLines := m.height
	if m.confirm != confirmNone {
		visibleLines -= 2 // reserve 2 for confirm bar (divider + message)
	}
	if visibleLines < 1 {
		visibleLines = 1
	}

	prevNotDone := false
	prevDone := false
	indent := strings.Repeat(" ", todoPrefixWidth)

	for i := m.scrollOff; i < len(m.todos) && linesUsed < visibleLines; i++ {
		todo := m.todos[i]

		// Blank line between consecutive non-done items, or at done→open transition.
		needGap := (todo.Status != model.StatusDone && prevNotDone) ||
			(todo.Status != model.StatusDone && prevDone)
		if needGap && linesUsed > 0 && linesUsed < visibleLines {
			b.WriteString("\n")
			linesUsed++
		}

		if linesUsed >= visibleLines {
			break
		}

		if linesUsed > 0 {
			b.WriteString("\n")
		}

		var prefix string
		switch todo.Status {
		case model.StatusDone:
			prefix = "✓ "
		case model.StatusInProgress:
			prefix = "▸ "
		default:
			prefix = "○ "
		}

		wrapped := wrapText(todo.Text, m.textWidth)
		selected := i == m.cursor && m.focused

		var style lipgloss.Style
		styled := false
		switch todo.Status {
		case model.StatusDone:
			style = todoDone
			styled = true
		case model.StatusInProgress:
			style = todoInProgress
			styled = true
		}
		if selected {
			style = style.Reverse(true)
			styled = true
		}

		for li, wl := range wrapped {
			if linesUsed >= visibleLines {
				break
			}
			var line string
			if li == 0 {
				line = prefix + wl
			} else {
				b.WriteString("\n")
				line = indent + wl
			}

			if styled {
				line = style.Render(line)
			}

			b.WriteString(line)
			linesUsed++
		}
		prevNotDone = todo.Status != model.StatusDone
		prevDone = todo.Status == model.StatusDone
	}

	// Assemble: pad items to fill visible area, then optional confirm bar.
	itemContent := b.String()
	itemLines := strings.Split(itemContent, "\n")
	for len(itemLines) < visibleLines {
		itemLines = append(itemLines, "")
	}
	itemLines = itemLines[:visibleLines]
	if m.confirm != confirmNone {
		itemLines = append(itemLines,
			dividerStyle.Render(strings.Repeat("─", m.width)),
			listConfirm.Render("Clear all AI tasks? (y to confirm)"),
		)
	}

	return strings.Join(itemLines, "\n")
}

func (m aiModel) reload() aiModel {
	todos, err := m.store.LoadAI()
	if err != nil {
		log.Printf("wtpad: failed to reload ai.md: %v", err)
		return m
	}
	m.todos = todos
	m = m.clampCursor()
	m = m.adjustScroll()
	return m
}

func (m aiModel) moveCursor(delta int) aiModel {
	if len(m.todos) == 0 {
		return m
	}
	newIdx := m.cursor + delta
	if newIdx < 0 || newIdx >= len(m.todos) {
		return m
	}
	m.cursor = newIdx
	m = m.adjustScroll()
	return m
}

func (m aiModel) clampCursor() aiModel {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if max := len(m.todos) - 1; m.cursor > max {
		if max < 0 {
			m.cursor = 0
		} else {
			m.cursor = max
		}
	}
	return m
}

func (m aiModel) availableLines() int {
	h := m.height
	if m.confirm != confirmNone {
		h -= 2
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (m aiModel) linesUpTo(targetIdx int) int {
	linesUsed := 0
	prevNotDone := false
	prevDone := false
	avail := m.availableLines()

	for i := m.scrollOff; i < len(m.todos) && i <= targetIdx; i++ {
		todo := m.todos[i]

		needGap := (todo.Status != model.StatusDone && prevNotDone) ||
			(todo.Status != model.StatusDone && prevDone)
		if needGap && linesUsed > 0 && linesUsed < avail {
			linesUsed++
		}

		linesUsed += len(wrapText(todo.Text, m.textWidth))
		prevNotDone = todo.Status != model.StatusDone
		prevDone = todo.Status == model.StatusDone
	}

	return linesUsed
}

func (m aiModel) adjustScroll() aiModel {
	if m.height < 1 || len(m.todos) == 0 {
		return m
	}

	if m.cursor < m.scrollOff {
		m.scrollOff = m.cursor
	}

	avail := m.availableLines()
	for m.linesUpTo(m.cursor) > avail {
		m.scrollOff++
		if m.scrollOff > m.cursor {
			m.scrollOff = m.cursor
			break
		}
	}

	for m.scrollOff > 0 && m.linesUpTo(len(m.todos)-1) < avail {
		m.scrollOff--
		if m.linesUpTo(m.cursor) > avail {
			m.scrollOff++
			break
		}
	}

	return m
}

// FooterHint returns the hint string for the footer bar.
func (m aiModel) FooterHint() string {
	return "? help · tab switch · q quit"
}

// count returns the number of AI todos.
func (m aiModel) count() int {
	return len(m.todos)
}

// StatusMsg returns the current transient status message (empty if none).
func (m aiModel) StatusMsg() string {
	return m.statusMsg
}

// HasItems reports whether the AI pane has any todos loaded.
func (m aiModel) HasItems() bool {
	return len(m.todos) > 0
}

// watchAIFile returns a tea.Cmd that watches the .wtpad/ directory for changes
// to ai.md. It sends aiFileChangedMsg on create/write/remove events.
// The watcher blocks until its event channel closes (on process exit).
// Returns nil if the watcher cannot be started.
func watchAIFile(dir string) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("wtpad: fsnotify watcher failed to start: %v", err)
			return nil
		}
		defer watcher.Close()

		if err := watcher.Add(dir); err != nil {
			log.Printf("wtpad: fsnotify cannot watch %s: %v", dir, err)
			return nil
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if filepath.Base(event.Name) != "ai.md" {
					continue
				}
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
					return aiFileChangedMsg{}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

// continueWatching re-starts the watcher after handling a file change event.
// This is needed because tea.Cmd is one-shot: after returning a message,
// the watcher goroutine exits. We re-launch it to keep watching.
func continueWatching(dir string) tea.Cmd {
	return watchAIFile(dir)
}
