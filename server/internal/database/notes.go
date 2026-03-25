package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/c0dev0id/notesd/server/internal/model"
)

func (db *DB) CreateNote(n *model.Note) error {
	_, err := db.sql.Exec(
		`INSERT INTO notes (id, user_id, title, content, type, modified_at, modified_by_device, deleted_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.UserID, n.Title, n.Content, n.Type,
		toMillis(n.ModifiedAt), n.ModifiedByDevice,
		toNullMillis(n.DeletedAt), toMillis(n.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	return nil
}

func (db *DB) GetNote(id, userID string) (*model.Note, error) {
	row := db.sql.QueryRow(
		`SELECT id, user_id, title, content, type, modified_at, modified_by_device, deleted_at, created_at
		 FROM notes WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, id, userID,
	)
	return scanNote(row)
}

// GetNoteAny returns a note regardless of soft-delete state. Used by sync.
func (db *DB) GetNoteAny(id, userID string) (*model.Note, error) {
	row := db.sql.QueryRow(
		`SELECT id, user_id, title, content, type, modified_at, modified_by_device, deleted_at, created_at
		 FROM notes WHERE id = ? AND user_id = ?`, id, userID,
	)
	return scanNote(row)
}

func (db *DB) ListNotes(userID string, limit, offset int) ([]model.Note, int, error) {
	var total int
	err := db.sql.QueryRow(
		`SELECT COUNT(*) FROM notes WHERE user_id = ? AND deleted_at IS NULL`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}

	rows, err := db.sql.Query(
		`SELECT id, user_id, title, content, type, modified_at, modified_by_device, deleted_at, created_at
		 FROM notes WHERE user_id = ? AND deleted_at IS NULL
		 ORDER BY modified_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	notes, err := scanNotes(rows)
	if err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

func (db *DB) UpdateNote(n *model.Note) error {
	res, err := db.sql.Exec(
		`UPDATE notes SET title = ?, content = ?, type = ?, modified_at = ?, modified_by_device = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		n.Title, n.Content, n.Type, toMillis(n.ModifiedAt), n.ModifiedByDevice,
		n.ID, n.UserID,
	)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	return checkRowsAffected(res)
}

func (db *DB) DeleteNote(id, userID string, deletedAt int64, deviceID string) error {
	res, err := db.sql.Exec(
		`UPDATE notes SET deleted_at = ?, modified_at = ?, modified_by_device = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		deletedAt, deletedAt, deviceID, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return checkRowsAffected(res)
}

func (db *DB) SearchNotes(userID, query string, limit, offset int) ([]model.Note, int, error) {
	pattern := "%" + query + "%"

	var total int
	err := db.sql.QueryRow(
		`SELECT COUNT(*) FROM notes
		 WHERE user_id = ? AND deleted_at IS NULL AND (title LIKE ? OR content LIKE ?)`,
		userID, pattern, pattern,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count search: %w", err)
	}

	rows, err := db.sql.Query(
		`SELECT id, user_id, title, content, type, modified_at, modified_by_device, deleted_at, created_at
		 FROM notes WHERE user_id = ? AND deleted_at IS NULL AND (title LIKE ? OR content LIKE ?)
		 ORDER BY modified_at DESC LIMIT ? OFFSET ?`,
		userID, pattern, pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	notes, err := scanNotes(rows)
	if err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

// GetNoteChangesSince returns all notes modified after the given timestamp (unix ms),
// including soft-deleted notes. Used by the sync endpoint.
func (db *DB) GetNoteChangesSince(userID string, sinceMs int64) ([]model.Note, error) {
	rows, err := db.sql.Query(
		`SELECT id, user_id, title, content, type, modified_at, modified_by_device, deleted_at, created_at
		 FROM notes WHERE user_id = ? AND modified_at > ?
		 ORDER BY modified_at ASC`,
		userID, sinceMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get note changes: %w", err)
	}
	defer rows.Close()
	return scanNotes(rows)
}

// UpsertNote inserts or updates a note using LWW conflict resolution.
// Returns the server's version if the incoming note loses the conflict.
func (db *DB) UpsertNote(n *model.Note) (*model.Note, error) {
	existing, err := db.GetNoteAny(n.ID, n.UserID)
	if errors.Is(err, ErrNotFound) {
		return nil, db.CreateNote(n)
	}
	if err != nil {
		return nil, err
	}

	// LWW: accept only if incoming timestamp is strictly newer
	if n.ModifiedAt.After(existing.ModifiedAt) {
		_, err := db.sql.Exec(
			`UPDATE notes SET title = ?, content = ?, type = ?, modified_at = ?,
			 modified_by_device = ?, deleted_at = ?
			 WHERE id = ? AND user_id = ?`,
			n.Title, n.Content, n.Type, toMillis(n.ModifiedAt),
			n.ModifiedByDevice, toNullMillis(n.DeletedAt),
			n.ID, n.UserID,
		)
		if err != nil {
			return nil, fmt.Errorf("upsert note: %w", err)
		}
		return nil, nil
	}

	// Server version wins — return it as conflict
	return existing, nil
}

func scanNote(row *sql.Row) (*model.Note, error) {
	var n model.Note
	var modifiedAt, createdAt int64
	var deletedAt sql.NullInt64
	err := row.Scan(
		&n.ID, &n.UserID, &n.Title, &n.Content, &n.Type,
		&modifiedAt, &n.ModifiedByDevice, &deletedAt, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan note: %w", err)
	}
	n.ModifiedAt = fromMillis(modifiedAt)
	n.DeletedAt = fromNullMillis(deletedAt)
	n.CreatedAt = fromMillis(createdAt)
	return &n, nil
}

func scanNotes(rows *sql.Rows) ([]model.Note, error) {
	var notes []model.Note
	for rows.Next() {
		var n model.Note
		var modifiedAt, createdAt int64
		var deletedAt sql.NullInt64
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Content, &n.Type,
			&modifiedAt, &n.ModifiedByDevice, &deletedAt, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan note row: %w", err)
		}
		n.ModifiedAt = fromMillis(modifiedAt)
		n.DeletedAt = fromNullMillis(deletedAt)
		n.CreatedAt = fromMillis(createdAt)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func checkRowsAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
