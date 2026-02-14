package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/c0dev0id/notesd/server/internal/database"
	"github.com/c0dev0id/notesd/server/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen  = 8
	maxPasswordLen  = 72 // bcrypt limit
	maxEmailLen     = 254
	maxDisplayName  = 200
)

const bcryptCost = 12

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "email, password, and display_name are required")
		return
	}

	if !isValidEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email address")
		return
	}
	if utf8.RuneCountInString(req.Email) > maxEmailLen {
		writeError(w, http.StatusBadRequest, "email too long")
		return
	}
	if utf8.RuneCountInString(req.Password) < minPasswordLen {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if len(req.Password) > maxPasswordLen {
		writeError(w, http.StatusBadRequest, "password too long")
		return
	}
	if utf8.RuneCountInString(req.DisplayName) > maxDisplayName {
		writeError(w, http.StatusBadRequest, "display name too long")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		slog.Error("bcrypt hash", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	now := model.NowMillis()
	user := &model.User{
		ID:           model.NewID(),
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  strings.TrimSpace(req.DisplayName),
		CreatedAt:    now,
	}

	if err := a.db.CreateUser(user); err != nil {
		if errors.Is(err, database.ErrConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("create user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" || req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "email, password, and device_id are required")
		return
	}

	user, err := a.db.GetUserByEmail(req.Email)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		slog.Error("get user by email", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	resp, err := a.issueTokenPair(user, req.DeviceID)
	if err != nil {
		slog.Error("issue token pair", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	userID, tokenID, deviceID, err := a.parseRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Look up stored token by hash
	tokenHash := database.HashToken(req.RefreshToken)
	stored, err := a.db.GetRefreshTokenByHash(tokenHash)
	if errors.Is(err, database.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "refresh token revoked")
		return
	}
	if err != nil {
		slog.Error("get refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if stored.ID != tokenID || stored.UserID != userID {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Rotation: delete old token
	if err := a.db.DeleteRefreshToken(stored.ID); err != nil {
		slog.Error("delete old refresh token", "error", err)
	}

	user, err := a.db.GetUserByID(userID)
	if err != nil {
		slog.Error("get user for refresh", "error", err)
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	resp, err := a.issueTokenPair(user, deviceID)
	if err != nil {
		slog.Error("issue token pair", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	if err := a.db.DeleteRefreshTokensByUser(userID); err != nil {
		slog.Error("delete refresh tokens on logout", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isValidEmail checks for a basic valid email format (has exactly one @, non-empty parts).
func isValidEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at < 1 {
		return false
	}
	domain := email[at+1:]
	if domain == "" || !strings.Contains(domain, ".") {
		return false
	}
	return true
}

// issueTokenPair creates both access and refresh tokens and stores the refresh token.
func (a *API) issueTokenPair(user *model.User, deviceID string) (*model.AuthResponse, error) {
	accessToken, err := a.issueAccessToken(user.ID, deviceID)
	if err != nil {
		return nil, err
	}

	tokenID := model.NewID()
	refreshToken, err := a.issueRefreshToken(tokenID, user.ID, deviceID)
	if err != nil {
		return nil, err
	}

	now := model.NowMillis()
	rt := &model.RefreshToken{
		ID:        tokenID,
		UserID:    user.ID,
		DeviceID:  deviceID,
		TokenHash: database.HashToken(refreshToken),
		ExpiresAt: now.Add(a.refreshTokenExpiry),
		CreatedAt: now,
	}
	if err := a.db.CreateRefreshToken(rt); err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}
