package api

import (
	"errors"
	"log/slog"
	"net/http"
	"unicode/utf8"

	"github.com/c0dev0id/notesd/server/internal/database"
	"github.com/c0dev0id/notesd/server/internal/model"
)

const (
	maxTitleLen   = 500
	maxContentLen = 500000 // 500KB of text
)

func (a *API) handleListNotes(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	if limit > 200 {
		limit = 200
	}

	notes, total, err := a.db.ListNotes(userID, limit, offset)
	if err != nil {
		slog.Error("list notes", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if notes == nil {
		notes = []model.Note{}
	}

	writeJSON(w, http.StatusOK, model.NoteListResponse{
		Notes:  notes,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (a *API) handleGetNote(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	id := r.PathValue("id")

	note, err := a.db.GetNote(id, userID)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		slog.Error("get note", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, note)
}

func (a *API) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())

	var req model.CreateNoteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	if utf8.RuneCountInString(req.Title) > maxTitleLen {
		writeError(w, http.StatusBadRequest, "title too long")
		return
	}
	if utf8.RuneCountInString(req.Content) > maxContentLen {
		writeError(w, http.StatusBadRequest, "content too long")
		return
	}

	noteType := req.Type
	if noteType == "" {
		noteType = "note"
	}
	if noteType != "note" && noteType != "todo_list" {
		writeError(w, http.StatusBadRequest, "type must be 'note' or 'todo_list'")
		return
	}

	now := model.NowMillis()
	note := &model.Note{
		ID:               model.NewID(),
		UserID:           userID,
		Title:            req.Title,
		Content:          req.Content,
		Type:             noteType,
		ModifiedAt:       now,
		ModifiedByDevice: req.DeviceID,
		CreatedAt:        now,
	}

	if err := a.db.CreateNote(note); err != nil {
		slog.Error("create note", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, note)
}

func (a *API) handleUpdateNote(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	id := r.PathValue("id")

	var req model.UpdateNoteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	if req.Title != nil && utf8.RuneCountInString(*req.Title) > maxTitleLen {
		writeError(w, http.StatusBadRequest, "title too long")
		return
	}
	if req.Content != nil && utf8.RuneCountInString(*req.Content) > maxContentLen {
		writeError(w, http.StatusBadRequest, "content too long")
		return
	}

	note, err := a.db.GetNote(id, userID)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		slog.Error("get note for update", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if req.Title != nil {
		note.Title = *req.Title
	}
	if req.Content != nil {
		note.Content = *req.Content
	}
	if req.Type != nil {
		if *req.Type != "note" && *req.Type != "todo_list" {
			writeError(w, http.StatusBadRequest, "type must be 'note' or 'todo_list'")
			return
		}
		note.Type = *req.Type
	}
	note.ModifiedAt = model.NowMillis()
	note.ModifiedByDevice = req.DeviceID

	if err := a.db.UpdateNote(note); err != nil {
		slog.Error("update note", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, note)
}

func (a *API) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	id := r.PathValue("id")
	deviceID := deviceIDFrom(r.Context())

	now := model.NowMillis().UnixMilli()
	err := a.db.DeleteNote(id, userID, now, deviceID)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		slog.Error("delete note", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleSearchNotes(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)
	if limit > 200 {
		limit = 200
	}

	notes, total, err := a.db.SearchNotes(userID, query, limit, offset)
	if err != nil {
		slog.Error("search notes", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if notes == nil {
		notes = []model.Note{}
	}

	writeJSON(w, http.StatusOK, model.NoteListResponse{
		Notes:  notes,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
