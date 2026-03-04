package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// Messages emitted by the template modal to the root app.
type importTemplateMsg struct{ todos []model.Todo }
type saveTemplateMsg struct{ name string }
type exitTemplateMsg struct{}

// enterTemplateMsg is sent by the todos pane to open the template modal.
type enterTemplateMsg struct{ saving bool }

// templateModalMode tracks the sub-state of the template modal.
type templateModalMode int

const (
	tmBrowse           templateModalMode = iota // browsing template list
	tmNameInput                                 // entering name for save-as
	tmConfirmOverwrite                          // confirming overwrite of existing template
)

type templateModal struct {
	mode      templateModalMode
	templates []store.TemplateInfo
	tstore    *store.TemplateStore
	cursor    int
	input     textinput.Model
	width     int
	height    int
	saving    bool // true = save-as flow, false = import flow
}

func newTemplateModal(ts *store.TemplateStore) templateModal {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 64
	ti.Placeholder = "template name"
	return templateModal{
		tstore: ts,
		input:  ti,
	}
}

// open initializes the modal for import or save-as.
func (m templateModal) open(saving bool, w, h int) templateModal {
	m.saving = saving
	m.width = w
	m.height = h
	m.cursor = 0

	templates, _ := m.tstore.ListTemplates()
	if templates == nil {
		templates = []store.TemplateInfo{}
	}
	m.templates = templates

	if saving {
		m.mode = tmNameInput
		m.input.SetValue("")
		m.input.Focus()
	} else {
		m.mode = tmBrowse
		m.input.Blur()
	}

	return m
}

func (m templateModal) resize(w, h int) templateModal {
	m.width = w
	m.height = h
	return m
}

func (m templateModal) Update(msg tea.Msg) (templateModal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch m.mode {
	case tmBrowse:
		return m.updateBrowse(keyMsg)
	case tmNameInput:
		return m.updateNameInput(msg)
	case tmConfirmOverwrite:
		return m.updateConfirmOverwrite(keyMsg)
	}
	return m, nil
}

func (m templateModal) updateBrowse(msg tea.KeyMsg) (templateModal, tea.Cmd) {
	switch msg.String() {
	case "up":
		if len(m.templates) > 0 {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.templates) - 1
			}
		}
	case "down":
		if len(m.templates) > 0 {
			m.cursor++
			if m.cursor >= len(m.templates) {
				m.cursor = 0
			}
		}
	case "enter":
		if len(m.templates) > 0 {
			tpl := m.templates[m.cursor]
			todos, err := m.tstore.LoadTemplate(tpl.Name)
			if err != nil {
				return m, nil
			}
			// Reset all imported todos to open status
			for i := range todos {
				todos[i].Status = model.StatusOpen
			}
			return m, func() tea.Msg { return importTemplateMsg{todos: todos} }
		}
	case "esc":
		return m, func() tea.Msg { return exitTemplateMsg{} }
	}
	return m, nil
}

func (m templateModal) updateNameInput(msg tea.Msg) (templateModal, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				return m, nil
			}
			if m.tstore.TemplateExists(name) {
				m.mode = tmConfirmOverwrite
				return m, nil
			}
			return m, func() tea.Msg { return saveTemplateMsg{name: name} }
		case "esc":
			return m, func() tea.Msg { return exitTemplateMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m templateModal) updateConfirmOverwrite(msg tea.KeyMsg) (templateModal, tea.Cmd) {
	switch msg.String() {
	case "y":
		name := strings.TrimSpace(m.input.Value())
		return m, func() tea.Msg { return saveTemplateMsg{name: name} }
	default:
		m.mode = tmNameInput
		return m, nil
	}
}

// FooterHint returns contextual hints for the app footer.
func (m templateModal) FooterHint() string {
	switch m.mode {
	case tmBrowse:
		if len(m.templates) == 0 {
			return "esc close"
		}
		return "↑/↓ navigate · enter select · esc cancel"
	case tmNameInput:
		return "enter confirm · esc cancel"
	case tmConfirmOverwrite:
		return "overwrite? y/n"
	}
	return ""
}

func (m templateModal) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	if m.saving {
		b.WriteString(templateHeader.Render("Save as Template"))
	} else {
		b.WriteString(templateHeader.Render("Import Template"))
	}
	b.WriteString("\n\n")

	switch m.mode {
	case tmBrowse:
		if len(m.templates) == 0 {
			b.WriteString(hintStyle.Render("No templates found."))
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("Save templates to ~/.wtpad/templates/"))
		} else {
			for i, tpl := range m.templates {
				label := fmt.Sprintf("%s (%d items)", tpl.Name, tpl.TodoCount)
				if i == m.cursor {
					b.WriteString(templateSelected.Render("▸ " + label))
				} else {
					b.WriteString("  " + label)
				}
				b.WriteString("\n")
			}
		}
	case tmNameInput:
		b.WriteString("Template name:\n")
		b.WriteString(m.input.View())
	case tmConfirmOverwrite:
		name := strings.TrimSpace(m.input.Value())
		b.WriteString(listConfirm.Render(fmt.Sprintf("Template %q already exists. Overwrite? (y/n)", name)))
	}

	content := b.String()

	contentWidth := lipgloss.Width(content)
	contentHeight := strings.Count(content, "\n") + 1

	padLeft := 0
	if m.width > contentWidth {
		padLeft = (m.width - contentWidth) / 2
	}
	padTop := 0
	if m.height > contentHeight {
		padTop = (m.height - contentHeight) / 2
	}

	return lipgloss.NewStyle().
		PaddingLeft(padLeft).
		PaddingTop(padTop).
		Width(m.width).
		Height(m.height).
		Render(content)
}
