package api

import (
	"errors"
	"log/slog"
	"net/http"
	"unicode/utf8"

	"github.com/c0dev0id/notesd/server/internal/database"
	"github.com/c0dev0id/notesd/server/internal/model"
)

const maxTodoContentLen = 10000

func (a *API) handleListTodos(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	if limit > 200 {
		limit = 200
	}

	todos, total, err := a.db.ListTodos(userID, limit, offset)
	if err != nil {
		slog.Error("list todos", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if todos == nil {
		todos = []model.Todo{}
	}

	writeJSON(w, http.StatusOK, model.TodoListResponse{
		Todos:  todos,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (a *API) handleGetTodo(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	id := r.PathValue("id")

	todo, err := a.db.GetTodo(id, userID)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		slog.Error("get todo", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, todo)
}

func (a *API) handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())

	var req model.CreateTodoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	if utf8.RuneCountInString(req.Content) > maxTodoContentLen {
		writeError(w, http.StatusBadRequest, "content too long")
		return
	}

	now := model.NowMillis()
	todo := &model.Todo{
		ID:               model.NewID(),
		UserID:           userID,
		NoteID:           req.NoteID,
		LineRef:          req.LineRef,
		Content:          req.Content,
		DueDate:          req.DueDate,
		Completed:        false,
		ModifiedAt:       now,
		ModifiedByDevice: req.DeviceID,
		CreatedAt:        now,
	}

	if err := a.db.CreateTodo(todo); err != nil {
		slog.Error("create todo", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, todo)
}

func (a *API) handleUpdateTodo(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	id := r.PathValue("id")

	var req model.UpdateTodoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	if req.Content != nil && utf8.RuneCountInString(*req.Content) > maxTodoContentLen {
		writeError(w, http.StatusBadRequest, "content too long")
		return
	}

	todo, err := a.db.GetTodo(id, userID)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		slog.Error("get todo for update", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if req.Content != nil {
		todo.Content = *req.Content
	}
	if req.DueDate != nil {
		todo.DueDate = req.DueDate
	}
	if req.Completed != nil {
		todo.Completed = *req.Completed
	}
	if req.NoteID != nil {
		todo.NoteID = req.NoteID
	}
	if req.LineRef != nil {
		todo.LineRef = req.LineRef
	}
	todo.ModifiedAt = model.NowMillis()
	todo.ModifiedByDevice = req.DeviceID

	if err := a.db.UpdateTodo(todo); err != nil {
		slog.Error("update todo", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, todo)
}

func (a *API) handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	id := r.PathValue("id")
	deviceID := deviceIDFrom(r.Context())

	now := model.NowMillis().UnixMilli()
	err := a.db.DeleteTodo(id, userID, now, deviceID)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		slog.Error("delete todo", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetOverdueTodos(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())

	todos, err := a.db.GetOverdueTodos(userID)
	if err != nil {
		slog.Error("get overdue todos", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if todos == nil {
		todos = []model.Todo{}
	}

	writeJSON(w, http.StatusOK, todos)
}
