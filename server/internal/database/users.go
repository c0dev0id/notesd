package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/c0dev0id/notesd/server/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

func (db *DB) CreateUser(u *model.User) error {
	_, err := db.sql.Exec(
		`INSERT INTO users (id, email, password_hash, display_name, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash, u.DisplayName, toMillis(u.CreatedAt),
	)
	if err != nil {
		// SQLite UNIQUE constraint on email
		if isConstraintError(err) {
			return fmt.Errorf("email already registered: %w", ErrConflict)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (db *DB) GetUserByID(id string) (*model.User, error) {
	row := db.sql.QueryRow(
		`SELECT id, email, password_hash, display_name, created_at
		 FROM users WHERE id = ?`, id,
	)
	return scanUser(row)
}

func (db *DB) GetUserByEmail(email string) (*model.User, error) {
	row := db.sql.QueryRow(
		`SELECT id, email, password_hash, display_name, created_at
		 FROM users WHERE email = ?`, email,
	)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	var createdAt int64
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.CreatedAt = fromMillis(createdAt)
	return &u, nil
}

// isConstraintError checks for SQLite constraint violations.
func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite returns error strings containing "UNIQUE constraint"
	return contains(err.Error(), "UNIQUE constraint") ||
		contains(err.Error(), "constraint failed")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
