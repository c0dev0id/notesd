package store

import (
	"os"
	"testing"
	"time"

	"github.com/c0dev0id/notesd/notes-cli/internal/model"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	f, err := os.CreateTemp("", "notes-cli-store-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(path) })

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

const testUser = "user-001"
const testDevice = "test-device"

// --- Note tests ---

func TestCreateAndGetNote(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	n := &model.Note{
		ID: model.NewID(), UserID: testUser,
		Title: "Hello", Content: "World", Type: "note",
		ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	if err := s.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	got, err := s.GetNote(n.ID, testUser)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	t.Logf("note: id=%s title=%q modified_at=%v", got.ID, got.Title, got.ModifiedAt)
	if got.Title != "Hello" || got.Content != "World" {
		t.Errorf("got title=%q content=%q", got.Title, got.Content)
	}
}

func TestGetNoteNotFound(t *testing.T) {
	s := openTestStore(t)
	_, err := s.GetNote("nonexistent", testUser)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	t.Log("ErrNotFound: ok")
}

func TestListNotes(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	for i := range 3 {
		n := &model.Note{
			ID: model.NewID(), UserID: testUser,
			Title: "Note", Type: "note",
			ModifiedAt: now.Add(time.Duration(i) * time.Second),
			ModifiedByDevice: testDevice, CreatedAt: now,
		}
		if err := s.CreateNote(n); err != nil {
			t.Fatalf("CreateNote %d: %v", i, err)
		}
	}
	notes, total, err := s.ListNotes(testUser, 10, 0)
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	t.Logf("list: total=%d returned=%d", total, len(notes))
	if total != 3 || len(notes) != 3 {
		t.Errorf("expected 3 notes, got total=%d len=%d", total, len(notes))
	}
}

func TestUpdateNote(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	n := &model.Note{
		ID: model.NewID(), UserID: testUser, Title: "Old", Type: "note",
		ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	if err := s.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	n.Title = "New"
	n.ModifiedAt = now.Add(time.Second)
	if err := s.UpdateNote(n); err != nil {
		t.Fatalf("UpdateNote: %v", err)
	}

	got, _ := s.GetNote(n.ID, testUser)
	t.Logf("updated note title: %q", got.Title)
	if got.Title != "New" {
		t.Errorf("expected title %q, got %q", "New", got.Title)
	}
}

func TestDeleteNoteSoftDeletes(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	n := &model.Note{
		ID: model.NewID(), UserID: testUser, Title: "ToDelete", Type: "note",
		ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	if err := s.CreateNote(n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := s.DeleteNote(n.ID, testUser, now.Add(time.Second).UnixMilli(), testDevice); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, err := s.GetNote(n.ID, testUser)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after soft-delete, got %v", err)
	}

	// GetNoteAny should still find it
	got, err := s.GetNoteAny(n.ID, testUser)
	if err != nil {
		t.Fatalf("GetNoteAny after delete: %v", err)
	}
	t.Logf("soft-deleted note: deleted_at=%v", got.DeletedAt)
	if got.DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}
}

func TestUpsertNoteLWW(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Seed
	n := &model.Note{
		ID: model.NewID(), UserID: testUser, Title: "Server", Type: "note",
		ModifiedAt: base, ModifiedByDevice: "dev-b", CreatedAt: base,
	}
	if err := s.CreateNote(n); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Older incoming — server wins
	older := *n
	older.Title = "Older Client"
	older.ModifiedAt = base.Add(-time.Second)
	winner, err := s.UpsertNote(&older)
	if err != nil {
		t.Fatalf("upsert older: %v", err)
	}
	t.Logf("older incoming: winner=%v (nil=incoming won)", winner)
	if winner == nil {
		t.Error("older incoming should not win")
	}

	// Newer incoming — client wins
	newer := *n
	newer.Title = "Newer Client"
	newer.ModifiedAt = base.Add(time.Second)
	winner, err = s.UpsertNote(&newer)
	if err != nil {
		t.Fatalf("upsert newer: %v", err)
	}
	if winner != nil {
		t.Error("newer incoming should win")
	}
	got, _ := s.GetNote(n.ID, testUser)
	t.Logf("after newer upsert: title=%q", got.Title)
	if got.Title != "Newer Client" {
		t.Errorf("expected newer title, got %q", got.Title)
	}

	// Equal timestamp, lower device — existing wins
	equal := *n
	equal.Title = "Lower Device"
	equal.ModifiedAt = newer.ModifiedAt
	equal.ModifiedByDevice = "aaa-low"
	winner, err = s.UpsertNote(&equal)
	if err != nil {
		t.Fatalf("upsert equal low: %v", err)
	}
	t.Logf("equal low device: winner=%v", winner)
	if winner == nil {
		t.Error("lower device should not win tie")
	}

	// Equal timestamp, higher device — incoming wins
	high := *n
	high.Title = "Higher Device"
	high.ModifiedAt = newer.ModifiedAt
	high.ModifiedByDevice = "zzz-high"
	winner, err = s.UpsertNote(&high)
	if err != nil {
		t.Fatalf("upsert equal high: %v", err)
	}
	if winner != nil {
		t.Error("higher device should win tie")
	}
	got, _ = s.GetNote(n.ID, testUser)
	t.Logf("after tiebreaker: title=%q device=%s", got.Title, got.ModifiedByDevice)
	if got.Title != "Higher Device" {
		t.Errorf("expected higher device title, got %q", got.Title)
	}
}

func TestNoteChangesSince(t *testing.T) {
	s := openTestStore(t)
	base := model.NowMillis()

	old := &model.Note{
		ID: model.NewID(), UserID: testUser, Title: "Old", Type: "note",
		ModifiedAt: base.Add(-2 * time.Hour), ModifiedByDevice: testDevice,
		CreatedAt: base.Add(-2 * time.Hour),
	}
	recent := &model.Note{
		ID: model.NewID(), UserID: testUser, Title: "Recent", Type: "note",
		ModifiedAt: base.Add(time.Minute), ModifiedByDevice: testDevice,
		CreatedAt: base.Add(time.Minute),
	}
	for _, n := range []*model.Note{old, recent} {
		if err := s.CreateNote(n); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	changes, err := s.GetNoteChangesSince(testUser, base.UnixMilli())
	if err != nil {
		t.Fatalf("GetNoteChangesSince: %v", err)
	}
	t.Logf("changes since now: %d (expected 1)", len(changes))
	if len(changes) != 1 || changes[0].Title != "Recent" {
		t.Errorf("expected 1 recent change, got %d", len(changes))
	}
}

func TestSearchNotes(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	for _, title := range []string{"Meeting notes", "Shopping list", "Meeting agenda"} {
		n := &model.Note{
			ID: model.NewID(), UserID: testUser, Title: title, Type: "note",
			ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
		}
		if err := s.CreateNote(n); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	results, total, err := s.SearchNotes(testUser, "Meeting", 10, 0)
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	t.Logf("search 'Meeting': total=%d results=%d", total, len(results))
	if total != 2 || len(results) != 2 {
		t.Errorf("expected 2 results, got total=%d len=%d", total, len(results))
	}
}

// --- Todo tests ---

func TestCreateAndGetTodo(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	td := &model.Todo{
		ID: model.NewID(), UserID: testUser, Content: "Buy milk",
		ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	if err := s.CreateTodo(td); err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}

	got, err := s.GetTodo(td.ID, testUser)
	if err != nil {
		t.Fatalf("GetTodo: %v", err)
	}
	t.Logf("todo: id=%s content=%q completed=%v", got.ID, got.Content, got.Completed)
	if got.Content != "Buy milk" || got.Completed {
		t.Errorf("got content=%q completed=%v", got.Content, got.Completed)
	}
}

func TestCompleteTodo(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	td := &model.Todo{
		ID: model.NewID(), UserID: testUser, Content: "Task",
		ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	if err := s.CreateTodo(td); err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}

	td.Completed = true
	td.ModifiedAt = now.Add(time.Second)
	if err := s.UpdateTodo(td); err != nil {
		t.Fatalf("UpdateTodo: %v", err)
	}

	got, _ := s.GetTodo(td.ID, testUser)
	t.Logf("completed todo: %v", got.Completed)
	if !got.Completed {
		t.Error("expected todo to be completed")
	}
}

func TestGetOverdueTodos(t *testing.T) {
	s := openTestStore(t)
	now := model.NowMillis()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	overdue := &model.Todo{
		ID: model.NewID(), UserID: testUser, Content: "Overdue",
		DueDate: &past, ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	upcoming := &model.Todo{
		ID: model.NewID(), UserID: testUser, Content: "Upcoming",
		DueDate: &future, ModifiedAt: now, ModifiedByDevice: testDevice, CreatedAt: now,
	}
	for _, td := range []*model.Todo{overdue, upcoming} {
		if err := s.CreateTodo(td); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	results, err := s.GetOverdueTodos(testUser)
	if err != nil {
		t.Fatalf("GetOverdueTodos: %v", err)
	}
	t.Logf("overdue todos: %d (expected 1)", len(results))
	if len(results) != 1 || results[0].Content != "Overdue" {
		t.Errorf("expected 1 overdue todo, got %d", len(results))
	}
}

// --- Sync state ---

func TestSyncStateRoundtrip(t *testing.T) {
	s := openTestStore(t)

	// Initial state
	got, err := s.GetLastSyncAt()
	if err != nil {
		t.Fatalf("GetLastSyncAt initial: %v", err)
	}
	t.Logf("initial last_sync_at: %d", got)
	if got != 0 {
		t.Errorf("expected 0 initially, got %d", got)
	}

	// Set
	ts := model.NowMillis().UnixMilli()
	if err := s.SetLastSyncAt(ts); err != nil {
		t.Fatalf("SetLastSyncAt: %v", err)
	}

	// Get back
	got, err = s.GetLastSyncAt()
	if err != nil {
		t.Fatalf("GetLastSyncAt after set: %v", err)
	}
	t.Logf("last_sync_at after set: %d", got)
	if got != ts {
		t.Errorf("expected %d, got %d", ts, got)
	}

	// Update
	ts2 := ts + 60000
	if err := s.SetLastSyncAt(ts2); err != nil {
		t.Fatalf("SetLastSyncAt update: %v", err)
	}
	got, _ = s.GetLastSyncAt()
	t.Logf("last_sync_at after update: %d", got)
	if got != ts2 {
		t.Errorf("expected %d after update, got %d", ts2, got)
	}
}
