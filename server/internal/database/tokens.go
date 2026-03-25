package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/c0dev0id/notesd/server/internal/model"
)

// HashToken produces a SHA-256 hex digest suitable for DB storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (db *DB) CreateRefreshToken(rt *model.RefreshToken) error {
	_, err := db.sql.Exec(
		`INSERT INTO refresh_tokens (id, user_id, device_id, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		rt.ID, rt.UserID, rt.DeviceID, rt.TokenHash,
		toMillis(rt.ExpiresAt), toMillis(rt.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (db *DB) GetRefreshTokenByHash(tokenHash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	var expiresAt, createdAt int64
	err := db.sql.QueryRow(
		`SELECT id, user_id, device_id, token_hash, expires_at, created_at
		 FROM refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.DeviceID, &rt.TokenHash, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	rt.ExpiresAt = fromMillis(expiresAt)
	rt.CreatedAt = fromMillis(createdAt)
	return &rt, nil
}

func (db *DB) DeleteRefreshToken(id string) error {
	_, err := db.sql.Exec(`DELETE FROM refresh_tokens WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

func (db *DB) DeleteRefreshTokensByUser(userID string) error {
	_, err := db.sql.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete user refresh tokens: %w", err)
	}
	return nil
}

func (db *DB) DeleteExpiredRefreshTokens() (int64, error) {
	now := model.NowMillis().UnixMilli()
	res, err := db.sql.Exec(`DELETE FROM refresh_tokens WHERE expires_at < ?`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}
	return res.RowsAffected()
}
