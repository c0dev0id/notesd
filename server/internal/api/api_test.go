package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/c0dev0id/notesd/server/internal/config"
	"github.com/c0dev0id/notesd/server/internal/database"
	"github.com/c0dev0id/notesd/server/internal/model"
)

// testSetup creates a test API server with an in-memory-like temp database.
type testEnv struct {
	api    *API
	server *httptest.Server
	db     *database.DB
}

func setup(t *testing.T) *testEnv {
	t.Helper()

	dbFile, err := os.CreateTemp("", "notesd-api-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	dbPath := dbFile.Name()
	dbFile.Close()
	t.Cleanup(func() { os.Remove(dbPath) })

	keyFile, err := os.CreateTemp("", "notesd-test-*.key")
	if err != nil {
		t.Fatalf("create temp key: %v", err)
	}
	keyPath := keyFile.Name()
	keyFile.Close()
	os.Remove(keyPath) // let API generate it
	t.Cleanup(func() { os.Remove(keyPath) })

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		Auth: config.AuthConfig{
			PrivateKeyPath:     keyPath,
			AccessTokenExpiry:  "15m",
			RefreshTokenExpiry: "720h",
		},
	}

	a, err := New(db, cfg)
	if err != nil {
		t.Fatalf("create api: %v", err)
	}

	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)

	return &testEnv{api: a, server: srv, db: db}
}

func (e *testEnv) doJSON(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

// registerAndLogin creates a user and returns the access token.
func (e *testEnv) registerAndLogin(t *testing.T) (string, *model.User) {
	t.Helper()
	email := fmt.Sprintf("test-%s@example.com", model.NewID()[:8])

	// Register
	resp := e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
		Email: email, Password: "testpass1234", DisplayName: "Test User",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("register: status=%d body=%s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Login
	resp = e.doJSON(t, "POST", "/api/v1/auth/login", model.LoginRequest{
		Email: email, Password: "testpass1234", DeviceID: "test-device",
	}, "")
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("login: status=%d body=%s", resp.StatusCode, body)
	}

	var authResp model.AuthResponse
	decodeBody(t, resp, &authResp)
	t.Logf("logged in: user_id=%s email=%s", authResp.User.ID, authResp.User.Email)
	return authResp.AccessToken, &authResp.User
}

// --- Auth tests ---

func TestRegister(t *testing.T) {
	e := setup(t)

	// Act
	resp := e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
		Email: "new@example.com", Password: "pass1234", DisplayName: "New User",
	}, "")

	// Assert
	t.Logf("register status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var user model.User
	decodeBody(t, resp, &user)
	t.Logf("registered user: id=%s email=%s display_name=%s", user.ID, user.Email, user.DisplayName)
	if user.Email != "new@example.com" {
		t.Errorf("email: got %q, want %q", user.Email, "new@example.com")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	e := setup(t)

	req := model.RegisterRequest{Email: "dup@example.com", Password: "password", DisplayName: "A"}
	resp := e.doJSON(t, "POST", "/api/v1/auth/register", req, "")
	resp.Body.Close()

	// Act
	resp = e.doJSON(t, "POST", "/api/v1/auth/register", req, "")

	// Assert
	t.Logf("duplicate register status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLoginSuccess(t *testing.T) {
	e := setup(t)

	// Arrange
	e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
		Email: "login@example.com", Password: "secret12", DisplayName: "Login User",
	}, "").Body.Close()

	// Act
	resp := e.doJSON(t, "POST", "/api/v1/auth/login", model.LoginRequest{
		Email: "login@example.com", Password: "secret12", DeviceID: "dev1",
	}, "")

	// Assert
	t.Logf("login status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var auth model.AuthResponse
	decodeBody(t, resp, &auth)
	t.Logf("access_token length: %d", len(auth.AccessToken))
	t.Logf("refresh_token length: %d", len(auth.RefreshToken))
	if auth.AccessToken == "" {
		t.Error("access_token is empty")
	}
	if auth.RefreshToken == "" {
		t.Error("refresh_token is empty")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	e := setup(t)

	e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
		Email: "wrong@example.com", Password: "correct1", DisplayName: "User",
	}, "").Body.Close()

	// Act
	resp := e.doJSON(t, "POST", "/api/v1/auth/login", model.LoginRequest{
		Email: "wrong@example.com", Password: "incorrec1", DeviceID: "dev1",
	}, "")

	// Assert
	t.Logf("wrong password status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRefreshToken(t *testing.T) {
	e := setup(t)

	// Arrange — register and login to get tokens
	e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
		Email: "refresh@example.com", Password: "password", DisplayName: "User",
	}, "").Body.Close()

	loginResp := e.doJSON(t, "POST", "/api/v1/auth/login", model.LoginRequest{
		Email: "refresh@example.com", Password: "password", DeviceID: "dev1",
	}, "")
	var loginAuth model.AuthResponse
	decodeBody(t, loginResp, &loginAuth)

	// Act — use refresh token
	resp := e.doJSON(t, "POST", "/api/v1/auth/refresh", model.RefreshRequest{
		RefreshToken: loginAuth.RefreshToken,
	}, "")

	// Assert
	t.Logf("refresh status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var newAuth model.AuthResponse
	decodeBody(t, resp, &newAuth)
	t.Logf("new access_token length: %d", len(newAuth.AccessToken))
	if newAuth.AccessToken == "" {
		t.Error("new access_token is empty")
	}
	if newAuth.RefreshToken == loginAuth.RefreshToken {
		t.Error("refresh token was not rotated")
	}

	// Old refresh token should be revoked
	resp = e.doJSON(t, "POST", "/api/v1/auth/refresh", model.RefreshRequest{
		RefreshToken: loginAuth.RefreshToken,
	}, "")
	t.Logf("reuse old refresh token status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for reused token, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestUnauthorizedAccess(t *testing.T) {
	e := setup(t)

	// Act — no token
	resp := e.doJSON(t, "GET", "/api/v1/notes", nil, "")

	// Assert
	t.Logf("no token status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Act — invalid token
	resp = e.doJSON(t, "GET", "/api/v1/notes", nil, "invalid.jwt.token")
	t.Logf("invalid token status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// --- Notes CRUD tests ---

func TestNoteCRUD(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Create
	resp := e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: "My Note", Content: "Hello", Type: "note", DeviceID: "dev1",
	}, token)
	t.Logf("create note status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	var note model.Note
	decodeBody(t, resp, &note)
	t.Logf("created note: id=%s title=%q", note.ID, note.Title)

	// Get
	resp = e.doJSON(t, "GET", "/api/v1/notes/"+note.ID, nil, token)
	t.Logf("get note status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got model.Note
	decodeBody(t, resp, &got)
	if got.Title != "My Note" {
		t.Errorf("title: got %q, want %q", got.Title, "My Note")
	}

	// Update
	newTitle := "Updated Note"
	resp = e.doJSON(t, "PUT", "/api/v1/notes/"+note.ID, model.UpdateNoteRequest{
		Title: &newTitle, DeviceID: "dev1",
	}, token)
	t.Logf("update note status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var updated model.Note
	decodeBody(t, resp, &updated)
	t.Logf("updated note: title=%q", updated.Title)
	if updated.Title != "Updated Note" {
		t.Errorf("title: got %q, want %q", updated.Title, "Updated Note")
	}

	// List
	resp = e.doJSON(t, "GET", "/api/v1/notes", nil, token)
	t.Logf("list notes status: %d", resp.StatusCode)
	var listResp model.NoteListResponse
	decodeBody(t, resp, &listResp)
	t.Logf("listed %d notes, total=%d", len(listResp.Notes), listResp.Total)
	if listResp.Total != 1 {
		t.Errorf("total: got %d, want 1", listResp.Total)
	}

	// Delete
	resp = e.doJSON(t, "DELETE", "/api/v1/notes/"+note.ID, nil, token)
	t.Logf("delete note status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Verify deleted
	resp = e.doJSON(t, "GET", "/api/v1/notes/"+note.ID, nil, token)
	t.Logf("get deleted note status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSearchNotes(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Arrange
	notes := []model.CreateNoteRequest{
		{Title: "Grocery List", Content: "milk and eggs", Type: "note", DeviceID: "dev1"},
		{Title: "Work Notes", Content: "quarterly review", Type: "note", DeviceID: "dev1"},
		{Title: "Recipes", Content: "need milk for cake", Type: "note", DeviceID: "dev1"},
	}
	for _, n := range notes {
		resp := e.doJSON(t, "POST", "/api/v1/notes", n, token)
		resp.Body.Close()
	}

	// Act
	resp := e.doJSON(t, "GET", "/api/v1/notes/search?q=milk", nil, token)

	// Assert
	t.Logf("search status: %d", resp.StatusCode)
	var result model.NoteListResponse
	decodeBody(t, resp, &result)
	t.Logf("search 'milk': %d results", result.Total)
	for _, n := range result.Notes {
		t.Logf("  - %q: %q", n.Title, n.Content)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 results, got %d", result.Total)
	}
}

// --- Todos CRUD tests ---

func TestTodoCRUD(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Create
	resp := e.doJSON(t, "POST", "/api/v1/todos", model.CreateTodoRequest{
		Content: "Buy groceries", DeviceID: "dev1",
	}, token)
	t.Logf("create todo status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	var todo model.Todo
	decodeBody(t, resp, &todo)
	t.Logf("created todo: id=%s content=%q", todo.ID, todo.Content)

	// Get
	resp = e.doJSON(t, "GET", "/api/v1/todos/"+todo.ID, nil, token)
	t.Logf("get todo status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got model.Todo
	decodeBody(t, resp, &got)
	if got.Content != "Buy groceries" {
		t.Errorf("content: got %q, want %q", got.Content, "Buy groceries")
	}

	// Update — mark completed
	completed := true
	resp = e.doJSON(t, "PUT", "/api/v1/todos/"+todo.ID, model.UpdateTodoRequest{
		Completed: &completed, DeviceID: "dev1",
	}, token)
	t.Logf("update todo status: %d", resp.StatusCode)
	var updatedTodo model.Todo
	decodeBody(t, resp, &updatedTodo)
	t.Logf("updated todo: completed=%v", updatedTodo.Completed)
	if !updatedTodo.Completed {
		t.Error("expected completed=true")
	}

	// Delete
	resp = e.doJSON(t, "DELETE", "/api/v1/todos/"+todo.ID, nil, token)
	t.Logf("delete todo status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestOverdueTodos(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Arrange — create overdue and future todos
	past := time.Now().UTC().Add(-24 * time.Hour)
	future := time.Now().UTC().Add(24 * time.Hour)

	e.doJSON(t, "POST", "/api/v1/todos", model.CreateTodoRequest{
		Content: "overdue", DueDate: &past, DeviceID: "dev1",
	}, token).Body.Close()
	e.doJSON(t, "POST", "/api/v1/todos", model.CreateTodoRequest{
		Content: "future", DueDate: &future, DeviceID: "dev1",
	}, token).Body.Close()

	// Act
	resp := e.doJSON(t, "GET", "/api/v1/todos/overdue", nil, token)

	// Assert
	t.Logf("overdue status: %d", resp.StatusCode)
	var todos []model.Todo
	decodeBody(t, resp, &todos)
	t.Logf("overdue todos: %d", len(todos))
	for _, td := range todos {
		t.Logf("  - %q due=%v", td.Content, td.DueDate)
	}
	if len(todos) != 1 {
		t.Errorf("expected 1 overdue, got %d", len(todos))
	}
}

// --- Sync tests ---

func TestSyncChanges(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Arrange — create a note
	since := time.Now().UTC().UnixMilli() - 1000
	e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: "Sync Me", Content: "content", Type: "note", DeviceID: "dev1",
	}, token).Body.Close()

	// Act
	resp := e.doJSON(t, "GET", fmt.Sprintf("/api/v1/sync/changes?since=%d", since), nil, token)

	// Assert
	t.Logf("sync changes status: %d", resp.StatusCode)
	var syncResp model.SyncChangesResponse
	decodeBody(t, resp, &syncResp)
	t.Logf("sync: %d notes, %d todos, timestamp=%d", len(syncResp.Notes), len(syncResp.Todos), syncResp.SyncTimestamp)
	if len(syncResp.Notes) != 1 {
		t.Errorf("expected 1 note change, got %d", len(syncResp.Notes))
	}
}

func TestSyncPush(t *testing.T) {
	e := setup(t)
	token, user := e.registerAndLogin(t)
	now := model.NowMillis()

	// Arrange — push a new note and todo
	noteID := model.NewID()
	todoID := model.NewID()
	pushReq := model.SyncPushRequest{
		Notes: []model.Note{
			{
				ID: noteID, UserID: user.ID,
				Title: "Pushed Note", Content: "from client",
				Type: "note", ModifiedAt: now, ModifiedByDevice: "phone",
				CreatedAt: now,
			},
		},
		Todos: []model.Todo{
			{
				ID: todoID, UserID: user.ID,
				Content: "Pushed Todo", Completed: false,
				ModifiedAt: now, ModifiedByDevice: "phone",
				CreatedAt: now,
			},
		},
		DeviceID: "phone",
	}

	// Act
	resp := e.doJSON(t, "POST", "/api/v1/sync/push", pushReq, token)

	// Assert
	t.Logf("sync push status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var pushResp model.SyncPushResponse
	decodeBody(t, resp, &pushResp)
	t.Logf("push result: accepted=%d conflicts=%d", pushResp.Accepted, len(pushResp.Conflicts))
	if pushResp.Accepted != 2 {
		t.Errorf("expected 2 accepted, got %d", pushResp.Accepted)
	}

	// Verify the pushed note exists
	resp = e.doJSON(t, "GET", "/api/v1/notes/"+noteID, nil, token)
	t.Logf("get pushed note status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected pushed note to exist, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSyncPushConflict(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Arrange — create a note via API (server has it)
	resp := e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: "Server Note", Content: "server version", Type: "note", DeviceID: "dev1",
	}, token)
	var serverNote model.Note
	decodeBody(t, resp, &serverNote)
	t.Logf("server note: id=%s modified_at=%v", serverNote.ID, serverNote.ModifiedAt)

	// Act — client pushes older version of the same note
	olderTime := serverNote.ModifiedAt.Add(-1 * time.Hour)
	pushReq := model.SyncPushRequest{
		Notes: []model.Note{
			{
				ID: serverNote.ID, UserID: serverNote.UserID,
				Title: "Client Note", Content: "client version",
				Type: "note", ModifiedAt: olderTime, ModifiedByDevice: "phone",
				CreatedAt: serverNote.CreatedAt,
			},
		},
	}
	resp = e.doJSON(t, "POST", "/api/v1/sync/push", pushReq, token)

	// Assert
	var pushResp model.SyncPushResponse
	decodeBody(t, resp, &pushResp)
	t.Logf("conflict push: accepted=%d conflicts=%d", pushResp.Accepted, len(pushResp.Conflicts))
	if len(pushResp.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(pushResp.Conflicts))
	}
	if len(pushResp.Conflicts) > 0 {
		c := pushResp.Conflicts[0]
		t.Logf("conflict: type=%s id=%s server_title=%q", c.Type, c.ID, c.ServerNote.Title)
		if c.ServerNote.Title != "Server Note" {
			t.Errorf("expected server version title %q, got %q", "Server Note", c.ServerNote.Title)
		}
	}

	// Server version should be preserved
	resp = e.doJSON(t, "GET", "/api/v1/notes/"+serverNote.ID, nil, token)
	var preserved model.Note
	decodeBody(t, resp, &preserved)
	t.Logf("preserved note: title=%q", preserved.Title)
	if preserved.Title != "Server Note" {
		t.Errorf("server version should win, got title %q", preserved.Title)
	}
}

// --- User isolation test ---

func TestUserIsolation(t *testing.T) {
	e := setup(t)
	token1, _ := e.registerAndLogin(t)
	token2, _ := e.registerAndLogin(t)

	// User 1 creates a note
	resp := e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: "Private Note", Content: "secret", Type: "note", DeviceID: "dev1",
	}, token1)
	var note model.Note
	decodeBody(t, resp, &note)
	t.Logf("user1 created note: id=%s", note.ID)

	// User 2 cannot see it
	resp = e.doJSON(t, "GET", "/api/v1/notes/"+note.ID, nil, token2)
	t.Logf("user2 get user1 note: status=%d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d — user isolation violated", resp.StatusCode)
	}
	resp.Body.Close()

	// User 2's list should be empty
	resp = e.doJSON(t, "GET", "/api/v1/notes", nil, token2)
	var listResp model.NoteListResponse
	decodeBody(t, resp, &listResp)
	t.Logf("user2 notes: total=%d", listResp.Total)
	if listResp.Total != 0 {
		t.Errorf("expected 0 notes for user2, got %d", listResp.Total)
	}
}

// --- Validation tests ---

func TestRegisterMissingFields(t *testing.T) {
	e := setup(t)

	tests := []struct {
		name string
		body model.RegisterRequest
	}{
		{"no email", model.RegisterRequest{Password: "longpass1", DisplayName: "A"}},
		{"no password", model.RegisterRequest{Email: "a@example.com", DisplayName: "A"}},
		{"no display_name", model.RegisterRequest{Email: "a@example.com", Password: "longpass1"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := e.doJSON(t, "POST", "/api/v1/auth/register", tc.body, "")
			t.Logf("%s: status=%d", tc.name, resp.StatusCode)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
			resp.Body.Close()
		})
	}
}

func TestRegisterInvalidEmail(t *testing.T) {
	e := setup(t)

	tests := []string{"notanemail", "missing@", "@nodomain", "no@dot"}
	for _, email := range tests {
		t.Run(email, func(t *testing.T) {
			resp := e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
				Email: email, Password: "longpass1", DisplayName: "User",
			}, "")
			t.Logf("email=%q status=%d", email, resp.StatusCode)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400 for email %q, got %d", email, resp.StatusCode)
			}
			resp.Body.Close()
		})
	}
}

func TestRegisterShortPassword(t *testing.T) {
	e := setup(t)

	resp := e.doJSON(t, "POST", "/api/v1/auth/register", model.RegisterRequest{
		Email: "short@example.com", Password: "short", DisplayName: "User",
	}, "")
	t.Logf("short password status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLoginMissingFields(t *testing.T) {
	e := setup(t)

	tests := []struct {
		name string
		body model.LoginRequest
	}{
		{"no email", model.LoginRequest{Password: "password", DeviceID: "d"}},
		{"no password", model.LoginRequest{Email: "a@b.com", DeviceID: "d"}},
		{"no device_id", model.LoginRequest{Email: "a@b.com", Password: "password"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := e.doJSON(t, "POST", "/api/v1/auth/login", tc.body, "")
			t.Logf("%s: status=%d", tc.name, resp.StatusCode)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
			resp.Body.Close()
		})
	}
}

func TestLoginNonExistentUser(t *testing.T) {
	e := setup(t)

	resp := e.doJSON(t, "POST", "/api/v1/auth/login", model.LoginRequest{
		Email: "ghost@example.com", Password: "password123", DeviceID: "dev1",
	}, "")
	t.Logf("non-existent user login status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLogout(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Act — logout
	resp := e.doJSON(t, "POST", "/api/v1/auth/logout", nil, token)
	t.Logf("logout status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRefreshTokenMissing(t *testing.T) {
	e := setup(t)

	resp := e.doJSON(t, "POST", "/api/v1/auth/refresh", model.RefreshRequest{}, "")
	t.Logf("empty refresh token status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRefreshTokenInvalid(t *testing.T) {
	e := setup(t)

	resp := e.doJSON(t, "POST", "/api/v1/auth/refresh", model.RefreshRequest{
		RefreshToken: "totally.invalid.token",
	}, "")
	t.Logf("invalid refresh token status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// --- Notes validation ---

func TestCreateNoteMissingDeviceID(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: "No Device", Content: "test", Type: "note",
	}, token)
	t.Logf("missing device_id status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCreateNoteInvalidType(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: "Bad Type", Type: "invalid_type", DeviceID: "dev1",
	}, token)
	t.Logf("invalid type status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetNoteNotFound(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "GET", "/api/v1/notes/nonexistent-id", nil, token)
	t.Logf("get non-existent note: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestUpdateNoteNotFound(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	title := "Updated"
	resp := e.doJSON(t, "PUT", "/api/v1/notes/nonexistent-id", model.UpdateNoteRequest{
		Title: &title, DeviceID: "dev1",
	}, token)
	t.Logf("update non-existent note: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestDeleteNoteNotFound(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "DELETE", "/api/v1/notes/nonexistent-id", nil, token)
	t.Logf("delete non-existent note: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSearchNotesEmptyQuery(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "GET", "/api/v1/notes/search", nil, token)
	t.Logf("empty search query status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestNoteTitleTooLong(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	longTitle := strings.Repeat("a", 501)
	resp := e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
		Title: longTitle, Type: "note", DeviceID: "dev1",
	}, token)
	t.Logf("long title status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// --- Todos validation ---

func TestCreateTodoMissingDeviceID(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "POST", "/api/v1/todos", model.CreateTodoRequest{
		Content: "No Device",
	}, token)
	t.Logf("missing device_id status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetTodoNotFound(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "GET", "/api/v1/todos/nonexistent-id", nil, token)
	t.Logf("get non-existent todo: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestDeleteTodoNotFound(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "DELETE", "/api/v1/todos/nonexistent-id", nil, token)
	t.Logf("delete non-existent todo: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestTodoUserIsolation(t *testing.T) {
	e := setup(t)
	token1, _ := e.registerAndLogin(t)
	token2, _ := e.registerAndLogin(t)

	// User 1 creates a todo
	resp := e.doJSON(t, "POST", "/api/v1/todos", model.CreateTodoRequest{
		Content: "Private Todo", DeviceID: "dev1",
	}, token1)
	var todo model.Todo
	decodeBody(t, resp, &todo)
	t.Logf("user1 created todo: id=%s", todo.ID)

	// User 2 cannot see it
	resp = e.doJSON(t, "GET", "/api/v1/todos/"+todo.ID, nil, token2)
	t.Logf("user2 get user1 todo: status=%d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d — todo isolation violated", resp.StatusCode)
	}
	resp.Body.Close()
}

// --- Sync validation ---

func TestSyncChangesMissingSince(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "GET", "/api/v1/sync/changes", nil, token)
	t.Logf("missing since status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSyncChangesInvalidSince(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "GET", "/api/v1/sync/changes?since=notanumber", nil, token)
	t.Logf("invalid since status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSyncPushEmptyBody(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	resp := e.doJSON(t, "POST", "/api/v1/sync/push", model.SyncPushRequest{}, token)
	t.Logf("empty push status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for empty push, got %d", resp.StatusCode)
	}

	var pushResp model.SyncPushResponse
	decodeBody(t, resp, &pushResp)
	t.Logf("empty push: accepted=%d conflicts=%d", pushResp.Accepted, len(pushResp.Conflicts))
	if pushResp.Accepted != 0 {
		t.Errorf("expected 0 accepted, got %d", pushResp.Accepted)
	}
}

// --- CORS test ---

func TestCORSPreflight(t *testing.T) {
	e := setup(t)

	req, _ := http.NewRequest("OPTIONS", e.server.URL+"/api/v1/notes", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("CORS preflight status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	allow := resp.Header.Get("Access-Control-Allow-Origin")
	t.Logf("Access-Control-Allow-Origin: %s", allow)
	if allow != "*" {
		t.Errorf("expected *, got %q", allow)
	}
}

// --- Health check test ---

func TestHealthCheck(t *testing.T) {
	e := setup(t)

	resp, err := http.Get(e.server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("health check status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var health map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	t.Logf("health: %v", health)
	if health["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", health["status"])
	}
}

// --- Pagination test ---

func TestNotesListPagination(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Create 5 notes
	for i := 0; i < 5; i++ {
		e.doJSON(t, "POST", "/api/v1/notes", model.CreateNoteRequest{
			Title: fmt.Sprintf("Note %d", i), Type: "note", DeviceID: "dev1",
		}, token).Body.Close()
	}

	// Request page with limit=2
	resp := e.doJSON(t, "GET", "/api/v1/notes?limit=2&offset=0", nil, token)
	var page1 model.NoteListResponse
	decodeBody(t, resp, &page1)
	t.Logf("page 1: %d notes, total=%d, limit=%d, offset=%d",
		len(page1.Notes), page1.Total, page1.Limit, page1.Offset)
	if page1.Total != 5 {
		t.Errorf("total: got %d, want 5", page1.Total)
	}
	if len(page1.Notes) != 2 {
		t.Errorf("page size: got %d, want 2", len(page1.Notes))
	}

	// Second page
	resp = e.doJSON(t, "GET", "/api/v1/notes?limit=2&offset=2", nil, token)
	var page2 model.NoteListResponse
	decodeBody(t, resp, &page2)
	t.Logf("page 2: %d notes", len(page2.Notes))
	if len(page2.Notes) != 2 {
		t.Errorf("page 2 size: got %d, want 2", len(page2.Notes))
	}
}

func TestNotesListLimitCap(t *testing.T) {
	e := setup(t)
	token, _ := e.registerAndLogin(t)

	// Request with limit > 200
	resp := e.doJSON(t, "GET", "/api/v1/notes?limit=999", nil, token)
	var listResp model.NoteListResponse
	decodeBody(t, resp, &listResp)
	t.Logf("capped limit: %d", listResp.Limit)
	if listResp.Limit != 200 {
		t.Errorf("expected limit capped to 200, got %d", listResp.Limit)
	}
}
