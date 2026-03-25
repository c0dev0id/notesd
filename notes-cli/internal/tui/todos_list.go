package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/c0dev0id/notesd/notes-cli/internal/model"
)

type todosListModel struct {
	todos    []model.Todo
	cursor   int
	total    int
	newInput string
	entering bool // typing a new todo inline
}

func newTodosListModel() todosListModel {
	return todosListModel{}
}

func (m *todosListModel) SetTodos(todos []model.Todo, total int) {
	m.todos = todos
	m.total = total
	if m.cursor >= len(todos) && len(todos) > 0 {
		m.cursor = len(todos) - 1
	}
}

func (m *todosListModel) Selected() *model.Todo {
	if len(m.todos) == 0 || m.cursor >= len(m.todos) {
		return nil
	}
	return &m.todos[m.cursor]
}

func (m *todosListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *todosListModel) MoveDown() {
	if m.cursor < len(m.todos)-1 {
		m.cursor++
	}
}

func (m todosListModel) View(width, height int) string {
	listHeight := height - 3

	if m.entering {
		input := styleSubtle.Render("New todo: ") + m.newInput + "█"
		if len(m.todos) == 0 {
			return styleSubtle.Render("No todos yet.") + "\n\n" + input
		}
	}

	if len(m.todos) == 0 {
		empty := styleSubtle.Render("No todos. Press 'n' to add one.")
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, empty)
	}

	end := listHeight
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
		end = start + listHeight
	}
	if end > len(m.todos) {
		end = len(m.todos)
	}

	var rows []string
	for i := start; i < end; i++ {
		t := m.todos[i]
		check := "[ ]"
		if t.Completed {
			check = styleSubtle.Render("[x]")
		}
		due := ""
		if t.DueDate != nil {
			due = styleSubtle.Render(" (" + t.DueDate.Local().Format("2006-01-02") + ")")
		}
		content := t.Content
		maxW := width - 6
		if len(content) > maxW {
			content = content[:maxW-1] + "…"
		}
		if t.Completed {
			content = styleSubtle.Render(content)
		}
		line := fmt.Sprintf("%s %s%s", check, content, due)
		if i == m.cursor {
			rows = append(rows, styleSelected.Width(width).Render(
				fmt.Sprintf("%s %s%s", check, t.Content, due),
			))
		} else {
			rows = append(rows, line)
		}
	}

	body := strings.Join(rows, "\n")
	if m.entering {
		input := styleSubtle.Render("New todo: ") + m.newInput + "█"
		body += "\n\n" + input
	}
	return body
}
