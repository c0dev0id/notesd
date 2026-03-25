package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type loginField int

const (
	loginFieldServer loginField = iota
	loginFieldEmail
	loginFieldPassword
)

type loginModel struct {
	fields  [3]textinput.Model
	active  loginField
	err     string
	loading bool
}

func newLoginModel() loginModel {
	server := textinput.New()
	server.Placeholder = "http://localhost:8080"
	server.Focus()

	email := textinput.New()
	email.Placeholder = "you@example.com"

	pass := textinput.New()
	pass.Placeholder = "password"
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'

	return loginModel{
		fields: [3]textinput.Model{server, email, pass},
		active: loginFieldServer,
	}
}

func (m *loginModel) serverURL() string {
	v := strings.TrimRight(m.fields[loginFieldServer].Value(), "/")
	if v == "" {
		v = m.fields[loginFieldServer].Placeholder
	}
	return v
}

func (m *loginModel) email() string { return m.fields[loginFieldEmail].Value() }
func (m *loginModel) password() string { return m.fields[loginFieldPassword].Value() }

func (m *loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	if m.loading {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown, tea.KeyEnter:
			if msg.Type == tea.KeyEnter && m.active == loginFieldPassword {
				// handled by parent
				return *m, nil
			}
			m.fields[m.active].Blur()
			m.active = (m.active + 1) % 3
			m.fields[m.active].Focus()
		case tea.KeyShiftTab, tea.KeyUp:
			m.fields[m.active].Blur()
			if m.active == 0 {
				m.active = loginField(len(m.fields) - 1)
			} else {
				m.active--
			}
			m.fields[m.active].Focus()
		}
	}

	var cmds [3]tea.Cmd
	for i := range m.fields {
		m.fields[i], cmds[i] = m.fields[i].Update(msg)
	}
	return *m, tea.Batch(cmds[:]...)
}

func (m loginModel) View(width int) string {
	labels := []string{"Server", "Email ", "Passw "}
	var rows []string
	for i, f := range m.fields {
		label := styleSubtle.Render(labels[i] + ": ")
		rows = append(rows, label+f.View())
	}

	form := strings.Join(rows, "\n")
	hint := styleHelp.Render("tab/shift-tab to move · enter to login · ctrl+c to quit")

	var errLine string
	if m.err != "" {
		errLine = "\n" + styleErr.Render("✗ "+m.err)
	}

	box := styleBorder.Width(width - 4).Render(
		styleTitle.Render("notes-tui login") + "\n\n" +
			form + errLine + "\n\n" + hint,
	)

	return lipgloss.Place(width, 0,
		lipgloss.Center, lipgloss.Top,
		box,
	)
}

// labelWidth for alignment
func labelWidth() int { return 8 }

// SetError sets the error message shown below the form.
func (m *loginModel) SetError(s string) { m.err = s }

// compact helpers
func (f loginField) String() string {
	return [...]string{"server", "email", "password"}[f]
}

func fmtField(label, val string) string {
	return fmt.Sprintf("%-8s %s", label+":", val)
}
