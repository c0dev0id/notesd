package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"crypto/rand"

	"github.com/c0dev0id/notesd/server/internal/config"
	"github.com/c0dev0id/notesd/server/internal/database"
	"github.com/c0dev0id/notesd/server/internal/model"
)

type API struct {
	db                 *database.DB
	config             *config.Config
	privateKey         *rsa.PrivateKey
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

func New(db *database.DB, cfg *config.Config) (*API, error) {
	key, err := loadOrGenerateKey(cfg.Auth.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}

	accessExp, err := time.ParseDuration(cfg.Auth.AccessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse access_token_expiry: %w", err)
	}
	refreshExp, err := time.ParseDuration(cfg.Auth.RefreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse refresh_token_expiry: %w", err)
	}

	return &API{
		db:                 db,
		config:             cfg,
		privateKey:         key,
		accessTokenExpiry:  accessExp,
		refreshTokenExpiry: refreshExp,
	}, nil
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public auth routes
	mux.HandleFunc("POST /api/v1/auth/register", a.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", a.handleRefresh)

	// Protected auth routes
	mux.HandleFunc("POST /api/v1/auth/logout", a.auth(a.handleLogout))

	// Notes
	mux.HandleFunc("GET /api/v1/notes/search", a.auth(a.handleSearchNotes))
	mux.HandleFunc("GET /api/v1/notes/{id}", a.auth(a.handleGetNote))
	mux.HandleFunc("GET /api/v1/notes", a.auth(a.handleListNotes))
	mux.HandleFunc("POST /api/v1/notes", a.auth(a.handleCreateNote))
	mux.HandleFunc("PUT /api/v1/notes/{id}", a.auth(a.handleUpdateNote))
	mux.HandleFunc("DELETE /api/v1/notes/{id}", a.auth(a.handleDeleteNote))

	// Todos
	mux.HandleFunc("GET /api/v1/todos/overdue", a.auth(a.handleGetOverdueTodos))
	mux.HandleFunc("GET /api/v1/todos/{id}", a.auth(a.handleGetTodo))
	mux.HandleFunc("GET /api/v1/todos", a.auth(a.handleListTodos))
	mux.HandleFunc("POST /api/v1/todos", a.auth(a.handleCreateTodo))
	mux.HandleFunc("PUT /api/v1/todos/{id}", a.auth(a.handleUpdateTodo))
	mux.HandleFunc("DELETE /api/v1/todos/{id}", a.auth(a.handleDeleteTodo))

	// Sync
	mux.HandleFunc("GET /api/v1/sync/changes", a.auth(a.handleSyncChanges))
	mux.HandleFunc("POST /api/v1/sync/push", a.auth(a.handleSyncPush))

	return logRequests(mux)
}

// Response helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return def
	}
	return v
}

// Request logging middleware

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// RSA key management

func loadOrGenerateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return parsePrivateKey(data)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	slog.Info("generating RSA key pair", "path", path)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}

	return key, nil
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
