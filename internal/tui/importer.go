package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type enterImportMsg struct{}
type importFileMsg struct {
	body string
}
type exitImportMsg struct{}

type importerModel struct {
	input  textinput.Model
	width  int
	height int
	errMsg string
}

func newImporterModel() importerModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "/path/to/file.md"
	ti.CharLimit = 512
	return importerModel{input: ti}
}

func (m importerModel) open(w, h int) importerModel {
	m.width = w
	m.height = h
	m.errMsg = ""
	m.input.SetValue("")
	m.input.Focus()
	return m
}

func (m importerModel) resize(w, h int) importerModel {
	m.width = w
	m.height = h
	return m
}

func (m importerModel) Update(msg tea.Msg) (importerModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch keyMsg.String() {
	case "enter":
		path := strings.TrimSpace(m.input.Value())
		if path == "" {
			return m, nil
		}
		path = expandTilde(path)
		path = strings.TrimSuffix(path, ".")
		if filepath.Ext(path) != ".md" {
			m.errMsg = "Only .md files can be imported"
			return m, nil
		}
		path = filepath.Clean(path)
		info, err := os.Stat(path)
		if err != nil {
			m.errMsg = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		if info.Size() > 1<<20 { // 1 MB
			m.errMsg = "File too large (max 1 MB)"
			return m, nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			m.errMsg = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		body := string(data)
		return m, func() tea.Msg { return importFileMsg{body: body} }
	case "esc":
		return m, func() tea.Msg { return exitImportMsg{} }
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m importerModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(modalHeader.Render("Import Note"))
	b.WriteString("\n\n")
	b.WriteString("File path:\n")
	b.WriteString(m.input.View())
	if m.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(listConfirm.Render(m.errMsg))
	}

	return renderCenteredContent(b.String(), m.width, m.height)
}

func (m importerModel) FooterHint() string {
	return "enter import · esc cancel"
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
