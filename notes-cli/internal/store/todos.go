package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/c0dev0id/notesd/notes-cli/internal/model"
)

func (s *Store) CreateTodo(t *model.Todo) error {
	_, err := s.db.Exec(
		`INSERT INTO todos
		 (id, user_id, note_id, line_ref, content, due_date, completed,
		  modified_at, modified_by_device, deleted_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.UserID, t.NoteID, t.LineRef, t.Content,
		toNullMillis(t.DueDate), t.Completed,
		toMillis(t.ModifiedAt), t.ModifiedByDevice,
		toNullMillis(t.DeletedAt), toMillis(t.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create todo: %w", err)
	}
	return nil
}

func (s *Store) GetTodo(id, userID string) (*model.Todo, error) {
	row := s.db.QueryRow(
		`SELECT id, user_id, note_id, line_ref, content, due_date, completed,
		 modified_at, modified_by_device, deleted_at, created_at
		 FROM todos WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, id, userID,
	)
	return scanTodo(row)
}

func (s *Store) GetTodoAny(id, userID string) (*model.Todo, error) {
	row := s.db.QueryRow(
		`SELECT id, user_id, note_id, line_ref, content, due_date, completed,
		 modified_at, modified_by_device, deleted_at, created_at
		 FROM todos WHERE id = ? AND user_id = ?`, id, userID,
	)
	return scanTodo(row)
}

func (s *Store) ListTodos(userID string, limit, offset int) ([]model.Todo, int, error) {
	var total int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM todos WHERE user_id = ? AND deleted_at IS NULL`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count todos: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, user_id, note_id, line_ref, content, due_date, completed,
		 modified_at, modified_by_device, deleted_at, created_at
		 FROM todos WHERE user_id = ? AND deleted_at IS NULL
		 ORDER BY modified_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list todos: %w", err)
	}
	defer rows.Close()
	todos, err := scanTodos(rows)
	return todos, total, err
}

func (s *Store) UpdateTodo(t *model.Todo) error {
	res, err := s.db.Exec(
		`UPDATE todos SET note_id = ?, line_ref = ?, content = ?, due_date = ?,
		 completed = ?, modified_at = ?, modified_by_device = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		t.NoteID, t.LineRef, t.Content, toNullMillis(t.DueDate),
		t.Completed, toMillis(t.ModifiedAt), t.ModifiedByDevice,
		t.ID, t.UserID,
	)
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}
	return checkRowsAffected(res)
}

func (s *Store) DeleteTodo(id, userID string, deletedAt int64, deviceID string) error {
	res, err := s.db.Exec(
		`UPDATE todos SET deleted_at = ?, modified_at = ?, modified_by_device = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		deletedAt, deletedAt, deviceID, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	return checkRowsAffected(res)
}

func (s *Store) GetOverdueTodos(userID string) ([]model.Todo, error) {
	now := model.NowMillis().UnixMilli()
	rows, err := s.db.Query(
		`SELECT id, user_id, note_id, line_ref, content, due_date, completed,
		 modified_at, modified_by_device, deleted_at, created_at
		 FROM todos
		 WHERE user_id = ? AND deleted_at IS NULL AND completed = 0
		   AND due_date IS NOT NULL AND due_date < ?
		 ORDER BY due_date ASC`,
		userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("get overdue todos: %w", err)
	}
	defer rows.Close()
	return scanTodos(rows)
}

// GetTodoChangesSince returns all todos (including deleted) modified after sinceMs.
func (s *Store) GetTodoChangesSince(userID string, sinceMs int64) ([]model.Todo, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, note_id, line_ref, content, due_date, completed,
		 modified_at, modified_by_device, deleted_at, created_at
		 FROM todos WHERE user_id = ? AND modified_at > ?
		 ORDER BY modified_at ASC`,
		userID, sinceMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get todo changes: %w", err)
	}
	defer rows.Close()
	return scanTodos(rows)
}

// UpsertTodo stores a todo using LWW conflict resolution.
func (s *Store) UpsertTodo(t *model.Todo) (*model.Todo, error) {
	existing, err := s.GetTodoAny(t.ID, t.UserID)
	if errors.Is(err, ErrNotFound) {
		return nil, s.CreateTodo(t)
	}
	if err != nil {
		return nil, err
	}

	if t.ModifiedAt.After(existing.ModifiedAt) ||
		(t.ModifiedAt.Equal(existing.ModifiedAt) && t.ModifiedByDevice > existing.ModifiedByDevice) {
		_, err := s.db.Exec(
			`UPDATE todos SET note_id = ?, line_ref = ?, content = ?, due_date = ?,
			 completed = ?, modified_at = ?, modified_by_device = ?, deleted_at = ?
			 WHERE id = ? AND user_id = ?`,
			t.NoteID, t.LineRef, t.Content, toNullMillis(t.DueDate),
			t.Completed, toMillis(t.ModifiedAt), t.ModifiedByDevice,
			toNullMillis(t.DeletedAt),
			t.ID, t.UserID,
		)
		if err != nil {
			return nil, fmt.Errorf("upsert todo: %w", err)
		}
		return nil, nil
	}

	return existing, nil
}

func scanTodo(row *sql.Row) (*model.Todo, error) {
	var t model.Todo
	var modifiedAt, createdAt int64
	var deletedAt, dueDate sql.NullInt64
	err := row.Scan(
		&t.ID, &t.UserID, &t.NoteID, &t.LineRef, &t.Content,
		&dueDate, &t.Completed,
		&modifiedAt, &t.ModifiedByDevice, &deletedAt, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan todo: %w", err)
	}
	t.ModifiedAt = fromMillis(modifiedAt)
	t.DeletedAt = fromNullMillis(deletedAt)
	t.DueDate = fromNullMillis(dueDate)
	t.CreatedAt = fromMillis(createdAt)
	return &t, nil
}

func scanTodos(rows *sql.Rows) ([]model.Todo, error) {
	var todos []model.Todo
	for rows.Next() {
		var t model.Todo
		var modifiedAt, createdAt int64
		var deletedAt, dueDate sql.NullInt64
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.NoteID, &t.LineRef, &t.Content,
			&dueDate, &t.Completed,
			&modifiedAt, &t.ModifiedByDevice, &deletedAt, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan todo row: %w", err)
		}
		t.ModifiedAt = fromMillis(modifiedAt)
		t.DeletedAt = fromNullMillis(deletedAt)
		t.DueDate = fromNullMillis(dueDate)
		t.CreatedAt = fromMillis(createdAt)
		todos = append(todos, t)
	}
	return todos, rows.Err()
}
