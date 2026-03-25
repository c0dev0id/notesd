// Package tui provides an interactive terminal UI for notes-tui.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/c0dev0id/notesd/notes-cli/internal/client"
	"github.com/c0dev0id/notesd/notes-cli/internal/model"
	"github.com/c0dev0id/notesd/notes-cli/internal/store"
	internalsync "github.com/c0dev0id/notesd/notes-cli/internal/sync"
)

// ---- messages ---------------------------------------------------------------

type syncDoneMsg struct {
	result *internalsync.Result
	err    error
}

type loginDoneMsg struct{ err error }
type tickMsg time.Time

// ---- screens ----------------------------------------------------------------

type screen int

const (
	screenNotesList screen = iota
	screenNoteEditor
	screenTodosList
	screenLogin
)

const syncInterval = 30 * time.Second

// ---- model ------------------------------------------------------------------

// Model is the root bubbletea model.
type Model struct {
	screen screen

	// dependencies
	cl     *client.Client
	st     *store.Store
	sy     *internalsync.Syncer
	userID string

	// sub-models
	login     loginModel
	notesList notesListModel
	editor    editorModel
	editingID string // empty = new note
	todos     todosListModel

	// UI state
	width   int
	height  int
	spinner spinner.Model
	syncing bool
	syncErr string
	status  string
}

// New creates a root model. If the client is not logged in, shows the login
// screen first.
func New(cl *client.Client, st *store.Store) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := &Model{
		cl:        cl,
		st:        st,
		spinner:   sp,
		notesList: newNotesListModel(),
		todos:     newTodosListModel(),
		login:     newLoginModel(),
	}

	if cl.IsLoggedIn() {
		m.userID = cl.SessionInfo().UserID
		m.sy = internalsync.New(st, cl, m.userID)
		m.screen = screenNotesList
	} else {
		m.screen = screenLogin
	}
	return m
}

// ---- Init -------------------------------------------------------------------

func (m *Model) Init() tea.Cmd {
	if m.screen == screenLogin {
		return nil
	}
	return tea.Batch(
		m.loadNotes(),
		m.doSync(),
		m.tick(),
	)
}

// ---- Update -----------------------------------------------------------------

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetSize(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+s":
			if m.screen == screenNoteEditor {
				return m, m.saveNote()
			}
		case "esc":
			switch m.screen {
			case screenNoteEditor:
				m.screen = screenNotesList
				return m, m.loadNotes()
			case screenTodosList:
				m.todos.entering = false
				m.screen = screenNotesList
				return m, m.loadNotes()
			}
		}
		return m.handleKey(msg)

	case loadNotesMsg:
		m.notesList.SetNotes(msg.notes, msg.total)
		return m, nil

	case loadTodosMsg:
		m.todos.SetTodos(msg.todos, msg.total)
		return m, nil

	case saveNoteMsg:
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
			return m, nil
		}
		m.screen = screenNotesList
		return m, tea.Batch(m.loadNotes(), m.doSync())

	case syncDoneMsg:
		m.syncing = false
		m.spinner, _ = m.spinner.Update(msg)
		if msg.err != nil {
			m.syncErr = msg.err.Error()
		} else {
			m.syncErr = ""
			// Reload current view after sync
			switch m.screen {
			case screenNotesList:
				return m, m.loadNotes()
			case screenTodosList:
				return m, m.loadTodos()
			}
		}
		return m, nil

	case loginDoneMsg:
		if msg.err != nil {
			m.login.SetError(msg.err.Error())
			return m, nil
		}
		m.userID = m.cl.SessionInfo().UserID
		m.sy = internalsync.New(m.st, m.cl, m.userID)
		m.screen = screenNotesList
		return m, tea.Batch(m.loadNotes(), m.doSync(), m.tick())

	case tickMsg:
		return m, tea.Batch(m.doSync(), m.tick())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Delegate to sub-models
	return m.updateSubModel(msg)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {

	case screenLogin:
		// Enter on last field → attempt login
		if msg.Type == tea.KeyEnter && m.login.active == loginFieldPassword {
			return m, m.doLogin()
		}
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd

	case screenNotesList:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "j", "down":
			m.notesList.MoveDown()
		case "k", "up":
			m.notesList.MoveUp()
		case "enter", "e":
			if n := m.notesList.Selected(); n != nil {
				m.editingID = n.ID
				m.editor = newEditorModel(n.Title, n.Content)
				m.editor.SetSize(m.width, m.height)
				m.screen = screenNoteEditor
			}
		case "n":
			m.editingID = ""
			m.editor = newEditorModel("", "")
			m.editor.SetSize(m.width, m.height)
			m.screen = screenNoteEditor
		case "d":
			if n := m.notesList.Selected(); n != nil {
				return m, m.deleteNote(n.ID)
			}
		case "t":
			m.screen = screenTodosList
			return m, m.loadTodos()
		case "s":
			return m, m.doSync()
		}
		return m, nil

	case screenNoteEditor:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd

	case screenTodosList:
		if m.todos.entering {
			switch msg.Type {
			case tea.KeyEnter:
				if m.todos.newInput != "" {
					return m, m.createTodo(m.todos.newInput)
				}
				m.todos.entering = false
			case tea.KeyEsc:
				m.todos.entering = false
				m.todos.newInput = ""
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.todos.newInput) > 0 {
					m.todos.newInput = m.todos.newInput[:len(m.todos.newInput)-1]
				}
			default:
				if msg.Type == tea.KeyRunes {
					m.todos.newInput += string(msg.Runes)
				}
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "esc":
			m.screen = screenNotesList
			return m, m.loadNotes()
		case "j", "down":
			m.todos.MoveDown()
		case "k", "up":
			m.todos.MoveUp()
		case "n":
			m.todos.entering = true
			m.todos.newInput = ""
		case " ":
			if t := m.todos.Selected(); t != nil {
				return m, m.toggleTodo(t)
			}
		case "d":
			if t := m.todos.Selected(); t != nil {
				return m, m.deleteTodo(t.ID)
			}
		case "s":
			return m, m.doSync()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateSubModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenLogin:
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd
	case screenNoteEditor:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}
	return m, nil
}

// ---- View -------------------------------------------------------------------

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	switch m.screen {
	case screenLogin:
		return m.login.View(m.width)
	case screenNotesList:
		return m.notesView()
	case screenNoteEditor:
		return m.editorView()
	case screenTodosList:
		return m.todosView()
	}
	return ""
}

func (m *Model) notesView() string {
	header := styleTitle.Render("Notes") + "  " +
		styleSubtle.Render("j/k: move  enter/e: open  n: new  d: delete  t: todos  s: sync  q: quit")
	body := m.notesList.View(m.width, m.height-2)
	status := m.statusLine()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m *Model) editorView() string {
	header := styleTitle.Render("Edit Note") + "  " +
		styleHelp.Render("ctrl+s: save  esc: discard")
	body := m.editor.View(m.width)
	status := m.statusLine()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m *Model) todosView() string {
	header := styleTitle.Render("Todos") + "  " +
		styleSubtle.Render("j/k: move  space: complete  n: new  d: delete  s: sync  esc: back")
	body := m.todos.View(m.width, m.height-2)
	status := m.statusLine()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m *Model) statusLine() string {
	if m.syncing {
		return styleStatusBar.Render(m.spinner.View() + " syncing…")
	}
	if m.syncErr != "" {
		return styleErr.Render("sync error: " + m.syncErr)
	}
	if m.status != "" {
		return styleStatusBar.Render(m.status)
	}
	return styleStatusBar.Render("─")
}

// ---- commands ---------------------------------------------------------------

type loadNotesMsg struct {
	notes []model.Note
	total int
}

type loadTodosMsg struct {
	todos []model.Todo
	total int
}

type saveNoteMsg struct{ err error }
type deleteMsg struct{ err error }

func (m *Model) loadNotes() tea.Cmd {
	return func() tea.Msg {
		notes, total, err := m.st.ListNotes(m.userID, 200, 0)
		if err != nil {
			return loadNotesMsg{}
		}
		return loadNotesMsg{notes: notes, total: total}
	}
}

func (m *Model) loadTodos() tea.Cmd {
	return func() tea.Msg {
		todos, total, err := m.st.ListTodos(m.userID, 200, 0)
		if err != nil {
			return loadTodosMsg{}
		}
		return loadTodosMsg{todos: todos, total: total}
	}
}

func (m *Model) saveNote() tea.Cmd {
	title := m.editor.Title()
	content := m.editor.Content()
	editingID := m.editingID
	userID := m.userID
	deviceID := m.cl.DeviceID()
	st := m.st

	return func() tea.Msg {
		now := model.NowMillis()
		if editingID == "" {
			n := &model.Note{
				ID: model.NewID(), UserID: userID,
				Title: title, Content: content, Type: "note",
				ModifiedAt: now, ModifiedByDevice: deviceID, CreatedAt: now,
			}
			return saveNoteMsg{err: st.CreateNote(n)}
		}
		n, err := st.GetNote(editingID, userID)
		if err != nil {
			return saveNoteMsg{err: err}
		}
		n.Title = title
		n.Content = content
		n.ModifiedAt = now
		n.ModifiedByDevice = deviceID
		return saveNoteMsg{err: st.UpdateNote(n)}
	}
}

func (m *Model) deleteNote(id string) tea.Cmd {
	userID := m.userID
	deviceID := m.cl.DeviceID()
	st := m.st

	return func() tea.Msg {
		now := model.NowMillis()
		err := st.DeleteNote(id, userID, now.UnixMilli(), deviceID)
		return deleteMsg{err: err}
	}
}

func (m *Model) createTodo(content string) tea.Cmd {
	userID := m.userID
	deviceID := m.cl.DeviceID()
	st := m.st

	return func() tea.Msg {
		now := model.NowMillis()
		t := &model.Todo{
			ID: model.NewID(), UserID: userID, Content: content,
			ModifiedAt: now, ModifiedByDevice: deviceID, CreatedAt: now,
		}
		if err := st.CreateTodo(t); err != nil {
			return loadTodosMsg{}
		}
		todos, total, _ := st.ListTodos(userID, 200, 0)
		return loadTodosMsg{todos: todos, total: total}
	}
}

func (m *Model) toggleTodo(t *model.Todo) tea.Cmd {
	userID := m.userID
	deviceID := m.cl.DeviceID()
	st := m.st
	id := t.ID
	completed := !t.Completed

	return func() tea.Msg {
		todo, err := st.GetTodo(id, userID)
		if err != nil {
			return loadTodosMsg{}
		}
		todo.Completed = completed
		todo.ModifiedAt = model.NowMillis()
		todo.ModifiedByDevice = deviceID
		st.UpdateTodo(todo)
		todos, total, _ := st.ListTodos(userID, 200, 0)
		return loadTodosMsg{todos: todos, total: total}
	}
}

func (m *Model) deleteTodo(id string) tea.Cmd {
	userID := m.userID
	deviceID := m.cl.DeviceID()
	st := m.st

	return func() tea.Msg {
		now := model.NowMillis()
		st.DeleteTodo(id, userID, now.UnixMilli(), deviceID)
		todos, total, _ := st.ListTodos(userID, 200, 0)
		return loadTodosMsg{todos: todos, total: total}
	}
}

func (m *Model) doSync() tea.Cmd {
	if m.sy == nil {
		return nil
	}
	m.syncing = true
	sy := m.sy
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			result, err := sy.Sync()
			return syncDoneMsg{result: result, err: err}
		},
	)
}

func (m *Model) doLogin() tea.Cmd {
	cl := m.cl
	serverURL := m.login.serverURL()
	email := m.login.email()
	password := m.login.password()

	return func() tea.Msg {
		err := cl.Login(serverURL, email, password, "notes-tui")
		return loginDoneMsg{err: err}
	}
}

func (m *Model) tick() tea.Cmd {
	return tea.Tick(syncInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Run starts the bubbletea program.
func Run(cl *client.Client, st *store.Store) error {
	m := New(cl, st)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
