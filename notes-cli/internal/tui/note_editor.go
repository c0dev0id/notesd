package tui

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type editorModel struct {
	titleInput textarea.Model
	body       textarea.Model
	focusBody  bool
	modified   bool
}

func newEditorModel(title, content string) editorModel {
	ti := textarea.New()
	ti.SetValue(title)
	ti.Placeholder = "Title"
	ti.ShowLineNumbers = false
	ti.SetHeight(1)
	ti.CharLimit = 255
	ti.Focus()

	body := textarea.New()
	body.SetValue(contentToText(content))
	body.Placeholder = "Write your note here…"
	body.ShowLineNumbers = false
	body.CharLimit = 0

	return editorModel{titleInput: ti, body: body}
}

func (m *editorModel) SetSize(width, height int) {
	// title row + separator (2) + help row (1)
	m.titleInput.SetWidth(width - 2)
	m.body.SetWidth(width - 2)
	m.body.SetHeight(height - 5)
}

func (m editorModel) Title() string   { return strings.TrimSpace(m.titleInput.Value()) }
func (m editorModel) Content() string { return m.body.Value() }

func (m *editorModel) Update(msg tea.Msg) (editorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if !m.focusBody {
				m.titleInput.Blur()
				m.body.Focus()
				m.focusBody = true
			}
			return *m, nil
		case "shift+tab":
			if m.focusBody {
				m.body.Blur()
				m.titleInput.Focus()
				m.focusBody = false
			}
			return *m, nil
		}
		m.modified = true
	}

	var cmd tea.Cmd
	if m.focusBody {
		m.body, cmd = m.body.Update(msg)
	} else {
		m.titleInput, cmd = m.titleInput.Update(msg)
	}
	return *m, cmd
}

func (m editorModel) View(width int) string {
	sep := styleSubtle.Render(strings.Repeat("─", width-2))

	hint := styleHelp.Render("tab: title↔body  ctrl+s: save  esc: discard")

	return lipgloss.JoinVertical(lipgloss.Left,
		m.titleInput.View(),
		sep,
		m.body.View(),
		hint,
	)
}

// contentToText extracts readable text from a Tiptap JSON doc, or returns
// the string as-is if it is not valid Tiptap JSON.
func contentToText(s string) string {
	if !strings.HasPrefix(strings.TrimSpace(s), "{") {
		return s
	}
	var doc struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(s), &doc); err != nil || doc.Type != "doc" {
		return s
	}
	var lines []string
	for _, block := range doc.Content {
		var parts []string
		for _, inline := range block.Content {
			if inline.Type == "text" {
				parts = append(parts, inline.Text)
			}
		}
		lines = append(lines, strings.Join(parts, ""))
	}
	return strings.Join(lines, "\n")
}
