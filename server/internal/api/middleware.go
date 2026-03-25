package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ctxUserID   contextKey = "user_id"
	ctxDeviceID contextKey = "device_id"
)

func userIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

func deviceIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxDeviceID).(string)
	return v
}

// auth wraps a handler with JWT access token verification.
func (a *API) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		if token == header {
			writeError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		claims := jwt.MapClaims{}
		parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return &a.privateKey.PublicKey, nil
		})
		if err != nil || !parsed.Valid {
			slog.Debug("jwt validation failed", "error", err)
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		tokenType, _ := claims["type"].(string)
		if tokenType != "access" {
			writeError(w, http.StatusUnauthorized, "invalid token type")
			return
		}

		sub, _ := claims["sub"].(string)
		deviceID, _ := claims["device_id"].(string)
		if sub == "" {
			writeError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, sub)
		ctx = context.WithValue(ctx, ctxDeviceID, deviceID)
		next(w, r.WithContext(ctx))
	}
}

// issueAccessToken creates a short-lived JWT access token.
func (a *API) issueAccessToken(userID, deviceID string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":       userID,
		"device_id": deviceID,
		"type":      "access",
		"iat":       now.Unix(),
		"exp":       now.Add(a.accessTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(a.privateKey)
}

// issueRefreshToken creates a long-lived JWT refresh token.
func (a *API) issueRefreshToken(tokenID, userID, deviceID string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":       userID,
		"jti":       tokenID,
		"device_id": deviceID,
		"type":      "refresh",
		"iat":       now.Unix(),
		"exp":       now.Add(a.refreshTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(a.privateKey)
}

// parseRefreshToken validates a refresh JWT and extracts claims.
func (a *API) parseRefreshToken(tokenStr string) (userID, tokenID, deviceID string, err error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &a.privateKey.PublicKey, nil
	})
	if err != nil || !parsed.Valid {
		return "", "", "", jwt.ErrSignatureInvalid
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return "", "", "", jwt.ErrSignatureInvalid
	}

	userID, _ = claims["sub"].(string)
	tokenID, _ = claims["jti"].(string)
	deviceID, _ = claims["device_id"].(string)
	return userID, tokenID, deviceID, nil
}
