package tui

import (
	"log"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// clearPromptStatusMsg clears the transient status message after a delay.
type clearPromptStatusMsg struct{}

func clearPromptStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearPromptStatusMsg{}
	})
}

// clipboardResultMsg carries the result of an async clipboard write.
type clipboardResultMsg struct{ err error }

const maxClipboardSize = 1 << 20 // 1 MB

type promptsModel struct {
	listPane
	store     *store.PromptStore
	statusMsg string
}

func newPrompts(prompts []model.Note, ps *store.PromptStore) promptsModel {
	m := promptsModel{
		listPane: listPane{items: notesToItems(prompts)},
		store:    ps,
	}
	m.listPane = m.listPane.loadBodies(m.loadBodyFn())
	return m
}

func (m promptsModel) SetSize(w, h int) promptsModel {
	m.listPane = m.listPane.setSize(w, h)
	return m
}

func (m promptsModel) SetFocus(focused bool) promptsModel {
	m.listPane = m.listPane.setFocus(focused)
	return m
}

func (m promptsModel) Update(msg tea.Msg) (promptsModel, tea.Cmd) {
	switch msg.(type) {
	case clearPromptStatusMsg:
		m.statusMsg = ""
		return m, nil
	case clipboardResultMsg:
		r := msg.(clipboardResultMsg)
		if r.err != nil {
			m.statusMsg = "Copy failed"
		} else {
			m.statusMsg = "Copied!"
		}
		return m, clearPromptStatusAfter(2 * time.Second)
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// Handle delete confirmation before any other key processing.
	if m.confirmDelete {
		if keyMsg.String() == "y" {
			m = m.deleteSelected()
		}
		m.confirmDelete = false
		return m, nil
	}

	// Prompts-specific key: clipboard copy
	if keyMsg.String() == "c" {
		if clipboard.Unsupported {
			m.statusMsg = "Clipboard not available"
			return m, clearPromptStatusAfter(2 * time.Second)
		}
		if item := m.selectedItem(); item != nil {
			body := item.Body
			if len(body) > maxClipboardSize {
				body = strings.ToValidUTF8(body[:maxClipboardSize], "")
				m.statusMsg = "Copying (truncated to 1 MB)…"
			} else {
				m.statusMsg = "Copying…"
			}
			return m, func() tea.Msg {
				return clipboardResultMsg{err: clipboard.WriteAll(body)}
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	var handled bool
	m.listPane, cmd, handled = m.listPane.handleKey(keyMsg)
	if handled {
		return m, cmd
	}

	return m, nil
}

func (m promptsModel) View() string {
	if len(m.items) == 0 {
		line1 := "Reusable text snippets."
		line2 := hintStyle.Render("Press 'a' to create your first prompt.")

		lines := []string{line1, line2}
		totalLines := len(lines)

		// Vertically center
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

	var barContent string
	if m.confirmDelete {
		barContent = listConfirm.Render("Delete prompt? (y to confirm)")
	} else {
		barContent = hintStyle.Render("Copy (c) · Add (a)")
	}

	return assembleListView(m.listPane, barContent)
}

func (m promptsModel) loadBodyFn() func(string) (string, error) {
	if m.store == nil {
		return nil
	}
	return func(name string) (string, error) {
		loaded, err := m.store.LoadPrompt(name)
		if err != nil {
			return "", err
		}
		return loaded.Body, nil
	}
}

func (m promptsModel) deleteSelected() promptsModel {
	if len(m.items) == 0 {
		return m
	}
	name := m.items[m.cursor].Name
	if err := m.store.DeletePrompt(name); err != nil {
		log.Printf("wtpad: failed to delete prompt %s: %v", name, err)
		return m
	}
	m.listPane = m.listPane.removeItem(m.cursor)
	return m
}

// SetPrompts replaces the prompts slice (used after editor saves a new/updated prompt).
func (m promptsModel) SetPrompts(prompts []model.Note) promptsModel {
	m.listPane = m.listPane.setItems(prompts, m.loadBodyFn())
	return m
}

// Init satisfies the tea.Model interface for standalone use.
func (m promptsModel) Init() tea.Cmd {
	return nil
}

// StatusMsg returns the current transient status message (empty if none).
func (m promptsModel) StatusMsg() string {
	return m.statusMsg
}

