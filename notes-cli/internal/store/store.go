package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	slog.Debug("store opened", "path", path)
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id                TEXT PRIMARY KEY,
			user_id           TEXT NOT NULL,
			title             TEXT NOT NULL DEFAULT '',
			content           TEXT NOT NULL DEFAULT '',
			type              TEXT NOT NULL DEFAULT 'note',
			modified_at       INTEGER NOT NULL,
			modified_by_device TEXT NOT NULL DEFAULT '',
			deleted_at        INTEGER,
			created_at        INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS todos (
			id                TEXT PRIMARY KEY,
			user_id           TEXT NOT NULL,
			note_id           TEXT,
			line_ref          TEXT,
			content           TEXT NOT NULL DEFAULT '',
			due_date          INTEGER,
			completed         INTEGER NOT NULL DEFAULT 0,
			modified_at       INTEGER NOT NULL,
			modified_by_device TEXT NOT NULL DEFAULT '',
			deleted_at        INTEGER,
			created_at        INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS sync_state (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_notes_user_modified
			ON notes(user_id, modified_at);
		CREATE INDEX IF NOT EXISTS idx_todos_user_modified
			ON todos(user_id, modified_at);
		CREATE INDEX IF NOT EXISTS idx_todos_due_date
			ON todos(due_date) WHERE due_date IS NOT NULL;
	`)
	return err
}

// timestamp helpers

func toMillis(t time.Time) int64 {
	return t.UnixMilli()
}

func fromMillis(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
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
	t := time.UnixMilli(n.Int64).UTC()
	return &t
}
