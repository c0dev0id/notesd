package database

import (
	"os"
	"testing"
	"time"

	"github.com/c0dev0id/notesd/server/internal/model"
)

// testDB creates a temporary database for testing and returns it with a cleanup function.
func testDB(t *testing.T) *DB {
	t.Helper()
	f, err := os.CreateTemp("", "notesd-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(path) })

	db, err := Open(path)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func testUser(t *testing.T, db *DB) *model.User {
	t.Helper()
	now := model.NowMillis()
	u := &model.User{
		ID:           model.NewID(),
		Email:        "test-" + model.NewID()[:8] + "@example.com",
		PasswordHash: "$2a$12$fakehashfakehashfakehashfakehashfakehashfakehashfake",
		DisplayName:  "Test User",
		CreatedAt:    now,
	}
	if err := db.CreateUser(u); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

// --- User tests ---

func TestCreateUser(t *testing.T) {
	db := testDB(t)

	// Arrange
	now := model.NowMillis()
	u := &model.User{
		ID:           model.NewID(),
		Email:        "alice@example.com",
		PasswordHash: "hash123",
		DisplayName:  "Alice",
		CreatedAt:    now,
	}

	// Act
	err := db.CreateUser(u)

	// Assert
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	t.Logf("created user id=%s email=%s", u.ID, u.Email)

	got, err := db.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	t.Logf("retrieved user: id=%s email=%s display_name=%s", got.ID, got.Email, got.DisplayName)

	if got.Email != u.Email {
		t.Errorf("email: got %q, want %q", got.Email, u.Email)
	}
	if got.DisplayName != u.DisplayName {
		t.Errorf("display_name: got %q, want %q", got.DisplayName, u.DisplayName)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	db := testDB(t)

	// Arrange
	now := model.NowMillis()
	u1 := &model.User{ID: model.NewID(), Email: "dup@example.com", PasswordHash: "h", DisplayName: "A", CreatedAt: now}
	u2 := &model.User{ID: model.NewID(), Email: "dup@example.com", PasswordHash: "h", DisplayName: "B", CreatedAt: now}

	// Act
	err := db.CreateUser(u1)
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	err = db.CreateUser(u2)

	// Assert
	t.Logf("duplicate email error: %v", err)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)

	// Act
	got, err := db.GetUserByEmail(u.Email)

	// Assert
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	t.Logf("found user by email: id=%s", got.ID)
	if got.ID != u.ID {
		t.Errorf("id: got %s, want %s", got.ID, u.ID)
	}
}

func TestGetUserNotFound(t *testing.T) {
	db := testDB(t)

	// Act
	_, err := db.GetUserByID("nonexistent")

	// Assert
	t.Logf("not found error: %v", err)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- Note tests ---

func TestCreateAndGetNote(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)

	// Arrange
	now := model.NowMillis()
	n := &model.Note{
		ID: model.NewID(), UserID: u.ID,
		Title: "Test Note", Content: "Hello world",
		Type: "note", ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}

	// Act
	if err := db.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	t.Logf("created note id=%s title=%q", n.ID, n.Title)

	got, err := db.GetNote(n.ID, u.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	t.Logf("retrieved note: id=%s title=%q content=%q type=%s", got.ID, got.Title, got.Content, got.Type)
	if got.Title != "Test Note" {
		t.Errorf("title: got %q, want %q", got.Title, "Test Note")
	}
	if got.Content != "Hello world" {
		t.Errorf("content: got %q, want %q", got.Content, "Hello world")
	}
}

func TestListNotes(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange — create 3 notes
	for i := 0; i < 3; i++ {
		n := &model.Note{
			ID: model.NewID(), UserID: u.ID,
			Title: "Note", Content: "", Type: "note",
			ModifiedAt: now.Add(time.Duration(i) * time.Millisecond),
			ModifiedByDevice: "dev1", CreatedAt: now,
		}
		if err := db.CreateNote(n); err != nil {
			t.Fatalf("create note %d: %v", i, err)
		}
	}

	// Act
	notes, total, err := db.ListNotes(u.ID, 10, 0)

	// Assert
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	t.Logf("listed %d notes, total=%d", len(notes), total)
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
	if len(notes) != 3 {
		t.Errorf("len: got %d, want 3", len(notes))
	}
}

func TestListNotesPagination(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	for i := 0; i < 5; i++ {
		n := &model.Note{
			ID: model.NewID(), UserID: u.ID,
			Title: "Note", Content: "", Type: "note",
			ModifiedAt: now.Add(time.Duration(i) * time.Millisecond),
			ModifiedByDevice: "dev1", CreatedAt: now,
		}
		if err := db.CreateNote(n); err != nil {
			t.Fatalf("create note %d: %v", i, err)
		}
	}

	// Act
	notes, total, err := db.ListNotes(u.ID, 2, 0)

	// Assert
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	t.Logf("page 1: %d notes, total=%d", len(notes), total)
	if total != 5 {
		t.Errorf("total: got %d, want 5", total)
	}
	if len(notes) != 2 {
		t.Errorf("page size: got %d, want 2", len(notes))
	}
}

func TestUpdateNote(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	n := &model.Note{
		ID: model.NewID(), UserID: u.ID,
		Title: "Original", Content: "old",
		Type: "note", ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}
	if err := db.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	// Act
	n.Title = "Updated"
	n.Content = "new"
	n.ModifiedAt = model.NowMillis()
	err := db.UpdateNote(n)

	// Assert
	if err != nil {
		t.Fatalf("UpdateNote: %v", err)
	}
	got, _ := db.GetNote(n.ID, u.ID)
	t.Logf("updated note: title=%q content=%q", got.Title, got.Content)
	if got.Title != "Updated" {
		t.Errorf("title: got %q, want %q", got.Title, "Updated")
	}
}

func TestDeleteNote(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	n := &model.Note{
		ID: model.NewID(), UserID: u.ID,
		Title: "Delete Me", Content: "",
		Type: "note", ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}
	if err := db.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	// Act — soft delete
	err := db.DeleteNote(n.ID, u.ID, model.NowMillis().UnixMilli(), "dev1")

	// Assert
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	_, err = db.GetNote(n.ID, u.ID)
	t.Logf("after delete, GetNote error: %v", err)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}

	// GetNoteAny should still find it
	got, err := db.GetNoteAny(n.ID, u.ID)
	if err != nil {
		t.Fatalf("GetNoteAny after delete: %v", err)
	}
	t.Logf("soft-deleted note still accessible via GetNoteAny: deleted_at=%v", got.DeletedAt)
	if got.DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}
}

func TestSearchNotes(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	notes := []struct{ title, content string }{
		{"Grocery List", "milk, eggs, bread"},
		{"Meeting Notes", "discuss project deadline"},
		{"Recipe", "chocolate cake with milk"},
	}
	for _, n := range notes {
		note := &model.Note{
			ID: model.NewID(), UserID: u.ID,
			Title: n.title, Content: n.content,
			Type: "note", ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
		}
		if err := db.CreateNote(note); err != nil {
			t.Fatalf("create note %q: %v", n.title, err)
		}
	}

	// Act
	results, total, err := db.SearchNotes(u.ID, "milk", 10, 0)

	// Assert
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	t.Logf("search 'milk': %d results, total=%d", len(results), total)
	for _, r := range results {
		t.Logf("  - %q: %q", r.Title, r.Content)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
}

// --- Todo tests ---

func TestCreateAndGetTodo(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	todo := &model.Todo{
		ID: model.NewID(), UserID: u.ID,
		Content: "Buy groceries", Completed: false,
		ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}

	// Act
	if err := db.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}
	t.Logf("created todo id=%s content=%q", todo.ID, todo.Content)

	got, err := db.GetTodo(todo.ID, u.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetTodo: %v", err)
	}
	t.Logf("retrieved todo: id=%s content=%q completed=%v", got.ID, got.Content, got.Completed)
	if got.Content != "Buy groceries" {
		t.Errorf("content: got %q, want %q", got.Content, "Buy groceries")
	}
	if got.Completed != false {
		t.Errorf("completed: got %v, want false", got.Completed)
	}
}

func TestTodoWithDueDate(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()
	due := now.Add(24 * time.Hour)

	// Arrange
	todo := &model.Todo{
		ID: model.NewID(), UserID: u.ID,
		Content: "Deadline task", DueDate: &due,
		ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}

	// Act
	if err := db.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}

	got, err := db.GetTodo(todo.ID, u.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetTodo: %v", err)
	}
	t.Logf("todo due_date: %v (expected: %v)", got.DueDate, due)
	if got.DueDate == nil {
		t.Fatal("due_date is nil, expected a value")
	}
	if got.DueDate.UnixMilli() != due.UnixMilli() {
		t.Errorf("due_date: got %v, want %v", got.DueDate, due)
	}
}

func TestUpdateTodo(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	todo := &model.Todo{
		ID: model.NewID(), UserID: u.ID,
		Content: "Original", Completed: false,
		ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}
	if err := db.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}

	// Act
	todo.Content = "Updated"
	todo.Completed = true
	todo.ModifiedAt = model.NowMillis()
	err := db.UpdateTodo(todo)

	// Assert
	if err != nil {
		t.Fatalf("UpdateTodo: %v", err)
	}
	got, _ := db.GetTodo(todo.ID, u.ID)
	t.Logf("updated todo: content=%q completed=%v", got.Content, got.Completed)
	if got.Content != "Updated" {
		t.Errorf("content: got %q, want %q", got.Content, "Updated")
	}
	if got.Completed != true {
		t.Errorf("completed: got %v, want true", got.Completed)
	}
}

func TestDeleteTodo(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	todo := &model.Todo{
		ID: model.NewID(), UserID: u.ID,
		Content: "Delete me", ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
	}
	if err := db.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}

	// Act
	err := db.DeleteTodo(todo.ID, u.ID, model.NowMillis().UnixMilli(), "dev1")

	// Assert
	if err != nil {
		t.Fatalf("DeleteTodo: %v", err)
	}
	_, err = db.GetTodo(todo.ID, u.ID)
	t.Logf("after delete, GetTodo error: %v", err)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetOverdueTodos(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange — one overdue, one future, one no due date, one completed overdue
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	todos := []struct {
		content   string
		dueDate   *time.Time
		completed bool
	}{
		{"overdue task", &past, false},
		{"future task", &future, false},
		{"no due date", nil, false},
		{"completed overdue", &past, true},
	}
	for _, td := range todos {
		todo := &model.Todo{
			ID: model.NewID(), UserID: u.ID,
			Content: td.content, DueDate: td.dueDate, Completed: td.completed,
			ModifiedAt: now, ModifiedByDevice: "dev1", CreatedAt: now,
		}
		if err := db.CreateTodo(todo); err != nil {
			t.Fatalf("create todo %q: %v", td.content, err)
		}
	}

	// Act
	overdue, err := db.GetOverdueTodos(u.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetOverdueTodos: %v", err)
	}
	t.Logf("overdue todos: %d", len(overdue))
	for _, td := range overdue {
		t.Logf("  - %q due=%v completed=%v", td.Content, td.DueDate, td.Completed)
	}
	if len(overdue) != 1 {
		t.Errorf("expected 1 overdue todo, got %d", len(overdue))
	}
	if len(overdue) > 0 && overdue[0].Content != "overdue task" {
		t.Errorf("expected 'overdue task', got %q", overdue[0].Content)
	}
}

// --- Sync tests ---

func TestNoteChangesSince(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange — create notes at different times
	t1 := now.Add(-2 * time.Hour)
	t2 := now.Add(-1 * time.Hour)
	t3 := now

	for i, ts := range []time.Time{t1, t2, t3} {
		n := &model.Note{
			ID: model.NewID(), UserID: u.ID,
			Title: "Note", Type: "note",
			ModifiedAt: ts, ModifiedByDevice: "dev1", CreatedAt: ts,
		}
		if err := db.CreateNote(n); err != nil {
			t.Fatalf("create note %d: %v", i, err)
		}
	}

	// Act — get changes since t1 (should exclude t1 itself)
	changes, err := db.GetNoteChangesSince(u.ID, t1.UnixMilli())

	// Assert
	if err != nil {
		t.Fatalf("GetNoteChangesSince: %v", err)
	}
	t.Logf("changes since %d: %d notes", t1.UnixMilli(), len(changes))
	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}
}

func TestUpsertNoteLWW(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange — create a note
	n := &model.Note{
		ID: model.NewID(), UserID: u.ID,
		Title: "Server Version", Content: "server content",
		Type: "note", ModifiedAt: now, ModifiedByDevice: "server", CreatedAt: now,
	}
	if err := db.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	// Act — client pushes older version (should lose)
	older := &model.Note{
		ID: n.ID, UserID: u.ID,
		Title: "Client Version", Content: "client content",
		Type: "note", ModifiedAt: now.Add(-1 * time.Hour), ModifiedByDevice: "client",
		CreatedAt: now,
	}
	conflict, err := db.UpsertNote(older)

	// Assert
	if err != nil {
		t.Fatalf("UpsertNote (older): %v", err)
	}
	t.Logf("older push: conflict=%v", conflict != nil)
	if conflict == nil {
		t.Error("expected conflict for older timestamp, got nil")
	}
	if conflict != nil {
		t.Logf("conflict server version: title=%q modified_at=%v", conflict.Title, conflict.ModifiedAt)
	}

	// Act — client pushes newer version (should win)
	newer := &model.Note{
		ID: n.ID, UserID: u.ID,
		Title: "Client Wins", Content: "new content",
		Type: "note", ModifiedAt: now.Add(1 * time.Hour), ModifiedByDevice: "client",
		CreatedAt: now,
	}
	conflict, err = db.UpsertNote(newer)

	// Assert
	if err != nil {
		t.Fatalf("UpsertNote (newer): %v", err)
	}
	t.Logf("newer push: conflict=%v", conflict != nil)
	if conflict != nil {
		t.Error("expected no conflict for newer timestamp")
	}

	got, _ := db.GetNote(n.ID, u.ID)
	t.Logf("final state: title=%q", got.Title)
	if got.Title != "Client Wins" {
		t.Errorf("title: got %q, want %q", got.Title, "Client Wins")
	}
}

// --- Refresh token tests ---

func TestRefreshTokenCRUD(t *testing.T) {
	db := testDB(t)
	u := testUser(t, db)
	now := model.NowMillis()

	// Arrange
	token := "random-token-string"
	hash := HashToken(token)
	rt := &model.RefreshToken{
		ID: model.NewID(), UserID: u.ID, DeviceID: "phone",
		TokenHash: hash, ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now,
	}

	// Act — create
	if err := db.CreateRefreshToken(rt); err != nil {
		t.Fatalf("CreateRefreshToken: %v", err)
	}
	t.Logf("created refresh token id=%s", rt.ID)

	// Act — lookup by hash
	got, err := db.GetRefreshTokenByHash(hash)
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash: %v", err)
	}
	t.Logf("found token: id=%s user_id=%s device_id=%s", got.ID, got.UserID, got.DeviceID)
	if got.ID != rt.ID {
		t.Errorf("id: got %s, want %s", got.ID, rt.ID)
	}

	// Act — delete
	if err := db.DeleteRefreshToken(rt.ID); err != nil {
		t.Fatalf("DeleteRefreshToken: %v", err)
	}
	_, err = db.GetRefreshTokenByHash(hash)
	t.Logf("after delete: %v", err)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
