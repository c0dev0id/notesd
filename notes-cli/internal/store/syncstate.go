package store

import (
	"database/sql"
	"errors"
	"strconv"
)

const keySyncAt = "last_sync_at"

// GetLastSyncAt returns the last successful sync timestamp in unix milliseconds.
// Returns 0 if no sync has occurred yet.
func (s *Store) GetLastSyncAt() (int64, error) {
	var val string
	err := s.db.QueryRow(
		`SELECT value FROM sync_state WHERE key = ?`, keySyncAt,
	).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// SetLastSyncAt records the most recent successful sync timestamp.
func (s *Store) SetLastSyncAt(ms int64) error {
	_, err := s.db.Exec(
		`INSERT INTO sync_state(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		keySyncAt, strconv.FormatInt(ms, 10),
	)
	return err
}
