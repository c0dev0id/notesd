package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite pragmas for performance and correctness
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := sqldb.Exec(p); err != nil {
			sqldb.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}

	db := &DB{sql: sqldb}
	if err := db.migrate(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	slog.Info("database opened", "path", path)
	return db, nil
}

func (db *DB) Close() error {
	return db.sql.Close()
}

func (db *DB) migrate() error {
	_, err := db.sql.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id           TEXT PRIMARY KEY,
	email        TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	display_name TEXT NOT NULL,
	created_at   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notes (
	id                TEXT PRIMARY KEY,
	user_id           TEXT NOT NULL REFERENCES users(id),
	title             TEXT NOT NULL DEFAULT '',
	content           TEXT NOT NULL DEFAULT '',
	type              TEXT NOT NULL DEFAULT 'note' CHECK(type IN ('note', 'todo_list')),
	modified_at       INTEGER NOT NULL,
	modified_by_device TEXT NOT NULL,
	deleted_at        INTEGER,
	created_at        INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
CREATE INDEX IF NOT EXISTS idx_notes_modified_at ON notes(modified_at);
CREATE INDEX IF NOT EXISTS idx_notes_deleted_at ON notes(deleted_at);

CREATE TABLE IF NOT EXISTS todos (
	id                TEXT PRIMARY KEY,
	user_id           TEXT NOT NULL REFERENCES users(id),
	note_id           TEXT REFERENCES notes(id),
	line_ref          TEXT,
	content           TEXT NOT NULL DEFAULT '',
	due_date          INTEGER,
	completed         INTEGER NOT NULL DEFAULT 0,
	modified_at       INTEGER NOT NULL,
	modified_by_device TEXT NOT NULL,
	deleted_at        INTEGER,
	created_at        INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id);
CREATE INDEX IF NOT EXISTS idx_todos_modified_at ON todos(modified_at);
CREATE INDEX IF NOT EXISTS idx_todos_deleted_at ON todos(deleted_at);
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);

CREATE TABLE IF NOT EXISTS refresh_tokens (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id),
	device_id  TEXT NOT NULL,
	token_hash TEXT NOT NULL,
	expires_at INTEGER NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
`

// Timestamp helpers for DB ↔ time.Time conversion.

func toMillis(t time.Time) int64 {
	return t.UnixMilli()
}

func fromMillis(ms int64) time.Time {
	return time.UnixMilli(ms)
}

func toNullMillis(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.UnixMilli(), Valid: true}
}

func fromNullMillis(n sql.NullInt64) *time.Time {
	if !n.Valid {
		return nil
	}
	t := time.UnixMilli(n.Int64)
	return &t
}
