// Package sync implements pull-then-push synchronisation between the local
// store and a notesd server. The algorithm:
//
//  1. Pull: fetch all server changes since last_sync_at, apply to local store
//     via LWW upsert.
//  2. Push: send all local items modified since last_sync_at to the server.
//     The server applies its own LWW upsert and returns any conflicts.
//  3. Resolve: for each conflict, apply the server's winning version to the
//     local store so both sides converge.
//  4. Record the sync timestamp returned by the server.
package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/c0dev0id/notesd/notes-cli/internal/model"
	"github.com/c0dev0id/notesd/notes-cli/internal/store"
)

// Client is the subset of the HTTP client used by Syncer.
type Client interface {
	DoJSON(method, path string, body, result any) (int, error)
	DeviceID() string
}

// Result summarises a completed sync operation.
type Result struct {
	NotesPulled    int
	NotesPushed    int
	NotesConflicts int
	TodosPulled    int
	TodosPushed    int
	TodosConflicts int
	ServerTime     time.Time
}

// Syncer holds the dependencies needed to run a sync.
type Syncer struct {
	store  *store.Store
	client Client
	userID string
}

func New(s *store.Store, c Client, userID string) *Syncer {
	return &Syncer{store: s, client: c, userID: userID}
}

// Sync runs a full pull-then-push cycle and returns a summary.
func (sy *Syncer) Sync() (*Result, error) {
	lastSync, err := sy.store.GetLastSyncAt()
	if err != nil {
		return nil, fmt.Errorf("get last sync: %w", err)
	}

	res := &Result{}

	// 1. Pull
	if err := sy.pull(lastSync, res); err != nil {
		return nil, fmt.Errorf("pull: %w", err)
	}

	// 2+3. Push and resolve conflicts
	if err := sy.push(lastSync, res); err != nil {
		return nil, fmt.Errorf("push: %w", err)
	}

	// 4. Record sync time
	if err := sy.store.SetLastSyncAt(res.ServerTime.UnixMilli()); err != nil {
		return nil, fmt.Errorf("set last sync: %w", err)
	}

	return res, nil
}

// --- server response types ---

type syncChangesResponse struct {
	Notes         []model.Note `json:"notes"`
	Todos         []model.Todo `json:"todos"`
	SyncTimestamp int64        `json:"sync_timestamp"`
}

type syncPushRequest struct {
	Notes []model.Note `json:"notes"`
	Todos []model.Todo `json:"todos"`
}

type syncConflict struct {
	Type       string      `json:"type"`
	ID         string      `json:"id"`
	ServerNote *model.Note `json:"server_note,omitempty"`
	ServerTodo *model.Todo `json:"server_todo,omitempty"`
}

type syncPushResponse struct {
	Accepted  int            `json:"accepted"`
	Conflicts []syncConflict `json:"conflicts"`
	Timestamp int64          `json:"timestamp"`
}

// pull fetches server changes and applies them to the local store.
func (sy *Syncer) pull(sinceMs int64, res *Result) error {
	var changes syncChangesResponse
	status, err := sy.client.DoJSON(
		"GET",
		fmt.Sprintf("/api/v1/sync/changes?since=%d", sinceMs),
		nil, &changes,
	)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("server returned %d", status)
	}

	for i := range changes.Notes {
		changes.Notes[i].UserID = sy.userID
		if _, err := sy.store.UpsertNote(&changes.Notes[i]); err != nil {
			return fmt.Errorf("upsert pulled note %s: %w", changes.Notes[i].ID, err)
		}
		res.NotesPulled++
	}
	for i := range changes.Todos {
		changes.Todos[i].UserID = sy.userID
		if _, err := sy.store.UpsertTodo(&changes.Todos[i]); err != nil {
			return fmt.Errorf("upsert pulled todo %s: %w", changes.Todos[i].ID, err)
		}
		res.TodosPulled++
	}

	res.ServerTime = time.UnixMilli(changes.SyncTimestamp).UTC()
	return nil
}

// push sends local changes to the server and resolves conflicts.
func (sy *Syncer) push(sinceMs int64, res *Result) error {
	notes, err := sy.store.GetNoteChangesSince(sy.userID, sinceMs)
	if err != nil {
		return err
	}
	todos, err := sy.store.GetTodoChangesSince(sy.userID, sinceMs)
	if err != nil {
		return err
	}

	if len(notes) == 0 && len(todos) == 0 {
		return nil
	}

	var pushResp syncPushResponse
	status, err := sy.client.DoJSON("POST", "/api/v1/sync/push",
		syncPushRequest{Notes: notes, Todos: todos}, &pushResp,
	)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("server returned %d on push", status)
	}

	res.NotesPushed = len(notes)
	res.TodosPushed = len(todos)

	// Resolve conflicts: apply server's winning version locally
	for _, c := range pushResp.Conflicts {
		switch c.Type {
		case "note":
			if c.ServerNote != nil {
				c.ServerNote.UserID = sy.userID
				if _, err := sy.store.UpsertNote(c.ServerNote); err != nil {
					return fmt.Errorf("resolve note conflict %s: %w", c.ID, err)
				}
				res.NotesConflicts++
			}
		case "todo":
			if c.ServerTodo != nil {
				c.ServerTodo.UserID = sy.userID
				if _, err := sy.store.UpsertTodo(c.ServerTodo); err != nil {
					return fmt.Errorf("resolve todo conflict %s: %w", c.ID, err)
				}
				res.TodosConflicts++
			}
		}
	}

	// Server time from push response supersedes pull time
	if pushResp.Timestamp > 0 {
		res.ServerTime = time.UnixMilli(pushResp.Timestamp).UTC()
	}
	return nil
}

// FormatResult returns a human-readable sync summary.
func FormatResult(r *Result) string {
	b, _ := json.MarshalIndent(map[string]any{
		"notes":  map[string]int{"pulled": r.NotesPulled, "pushed": r.NotesPushed, "conflicts": r.NotesConflicts},
		"todos":  map[string]int{"pulled": r.TodosPulled, "pushed": r.TodosPushed, "conflicts": r.TodosConflicts},
		"server_time": r.ServerTime.Format(time.RFC3339),
	}, "", "  ")
	return string(b)
}
