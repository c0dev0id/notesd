package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/c0dev0id/notesd/notes-cli/internal/model"
)

type notesListModel struct {
	notes    []model.Note
	cursor   int
	offset   int
	total    int
	pageSize int
}

func newNotesListModel() notesListModel {
	return notesListModel{pageSize: 20}
}

func (m *notesListModel) SetNotes(notes []model.Note, total int) {
	m.notes = notes
	m.total = total
	if m.cursor >= len(notes) && len(notes) > 0 {
		m.cursor = len(notes) - 1
	}
}

func (m *notesListModel) Selected() *model.Note {
	if len(m.notes) == 0 || m.cursor >= len(m.notes) {
		return nil
	}
	return &m.notes[m.cursor]
}

func (m *notesListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *notesListModel) MoveDown() {
	if m.cursor < len(m.notes)-1 {
		m.cursor++
	}
}

func (m notesListModel) View(width, height int) string {
	if len(m.notes) == 0 {
		empty := styleSubtle.Render("No notes. Press 'n' to create one.")
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, empty)
	}

	// Leave room for header (2) and status bar (1)
	listHeight := height - 3
	if listHeight < 1 {
		listHeight = 1
	}

	// Scroll window
	start := m.cursor - listHeight/2
	if start < 0 {
		start = 0
	}
	end := start + listHeight
	if end > len(m.notes) {
		end = len(m.notes)
		start = end - listHeight
		if start < 0 {
			start = 0
		}
	}

	idW := 8  // show first 8 chars of ID
	typeW := 10
	dateW := 10
	titleW := width - idW - typeW - dateW - 6

	header := styleSubtle.Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			idW, "ID", typeW, "TYPE", dateW, "MODIFIED", "TITLE"),
	)

	var rows []string
	for i := start; i < end; i++ {
		n := m.notes[i]
		title := n.Title
		if title == "" {
			title = "(untitled)"
		}
		if len(title) > titleW {
			title = title[:titleW-1] + "…"
		}
		shortID := n.ID
		if len(shortID) > idW {
			shortID = shortID[:idW]
		}
		noteType := n.Type
		if len(noteType) > typeW {
			noteType = noteType[:typeW]
		}
		date := n.ModifiedAt.Local().Format("2006-01-02")
		line := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			idW, shortID, typeW, noteType, dateW, date, title)
		if i == m.cursor {
			rows = append(rows, styleSelected.Width(width).Render(line))
		} else {
			rows = append(rows, line)
		}
	}

	pagination := ""
	if m.total > len(m.notes) {
		pagination = styleSubtle.Render(
			fmt.Sprintf("  %d-%d of %d", m.offset+1, m.offset+len(m.notes), m.total),
		)
	}

	return header + "\n" + strings.Join(rows, "\n") + pagination
}
