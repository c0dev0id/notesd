package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/c0dev0id/notesd/server/internal/model"
)

func (a *API) handleSyncChanges(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())

	sinceStr := r.URL.Query().Get("since")
	if sinceStr == "" {
		writeError(w, http.StatusBadRequest, "since parameter is required")
		return
	}

	sinceMs, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "since must be a unix timestamp in milliseconds")
		return
	}

	notes, err := a.db.GetNoteChangesSince(userID, sinceMs)
	if err != nil {
		slog.Error("get note changes", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if notes == nil {
		notes = []model.Note{}
	}

	todos, err := a.db.GetTodoChangesSince(userID, sinceMs)
	if err != nil {
		slog.Error("get todo changes", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if todos == nil {
		todos = []model.Todo{}
	}

	writeJSON(w, http.StatusOK, model.SyncChangesResponse{
		Notes:         notes,
		Todos:         todos,
		SyncTimestamp: model.NowMillis().UnixMilli(),
	})
}

func (a *API) handleSyncPush(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())

	var req model.SyncPushRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var conflicts []model.SyncConflict
	accepted := 0

	for i := range req.Notes {
		req.Notes[i].UserID = userID
		serverVersion, err := a.db.UpsertNote(&req.Notes[i])
		if err != nil {
			slog.Error("sync upsert note", "id", req.Notes[i].ID, "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if serverVersion != nil {
			conflicts = append(conflicts, model.SyncConflict{
				Type:       "note",
				ID:         req.Notes[i].ID,
				ServerNote: serverVersion,
			})
		} else {
			accepted++
		}
	}

	for i := range req.Todos {
		req.Todos[i].UserID = userID
		serverVersion, err := a.db.UpsertTodo(&req.Todos[i])
		if err != nil {
			slog.Error("sync upsert todo", "id", req.Todos[i].ID, "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if serverVersion != nil {
			conflicts = append(conflicts, model.SyncConflict{
				Type:       "todo",
				ID:         req.Todos[i].ID,
				ServerTodo: serverVersion,
			})
		} else {
			accepted++
		}
	}

	writeJSON(w, http.StatusOK, model.SyncPushResponse{
		Conflicts: conflicts,
		Accepted:  accepted,
		Timestamp: model.NowMillis().UnixMilli(),
	})
}
