package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

// newTestClient creates a Client pointing at srv with a temp config directory.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("NewWithDir: %v", err)
	}
	if srv != nil {
		c.BaseURL = srv.URL
	}
	c.DeviceID = "test-device"
	return c
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func authResp(userID, email, display string) AuthResponse {
	r := AuthResponse{AccessToken: "access-tok", RefreshToken: "refresh-tok"}
	r.User.ID = userID
	r.User.Email = email
	r.User.DisplayName = display
	return r
}

// --- Login ---

func TestLoginSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, authResp("uid1", "user@example.com", "User"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Login(srv.URL, "user@example.com", "pass", "dev1"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	t.Logf("session: user=%s email=%s", c.session.UserID, c.session.Email)

	if !c.IsLoggedIn() {
		t.Error("expected IsLoggedIn to be true")
	}
	if c.session.AccessToken != "access-tok" {
		t.Errorf("access token: got %q", c.session.AccessToken)
	}
	if c.session.Email != "user@example.com" {
		t.Errorf("email: got %q", c.session.Email)
	}

	// Session must be persisted to disk
	s2, err := c.loadSession()
	if err != nil {
		t.Fatalf("loadSession after login: %v", err)
	}
	if s2.AccessToken != "access-tok" {
		t.Errorf("persisted access token: got %q", s2.AccessToken)
	}
	t.Logf("persisted session: access_token=%s refresh_token=%s", s2.AccessToken, s2.RefreshToken)
}

func TestLoginWrongPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Login(srv.URL, "user@example.com", "wrong", "dev1")
	t.Logf("login error: %v", err)
	if err == nil {
		t.Error("expected error on wrong password")
	}
	if c.IsLoggedIn() {
		t.Error("should not be logged in after failed login")
	}
}

// --- Register ---

func TestRegisterSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Register(srv.URL, "new@example.com", "pass1234", "New User"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	t.Log("register: success")
}

func TestRegisterDuplicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Register(srv.URL, "dup@example.com", "pass1234", "Dup")
	t.Logf("duplicate register error: %v", err)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

// --- Logout ---

func TestLogoutClearsSession(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/logout" {
			called = true
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.session = &Session{AccessToken: "tok", RefreshToken: "ref", ServerURL: srv.URL}
	_ = c.saveSession()

	if err := c.Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	t.Logf("logout called server: %v", called)
	if !called {
		t.Error("expected logout endpoint to be called")
	}
	if c.IsLoggedIn() {
		t.Error("should not be logged in after logout")
	}

	// Session file must be gone
	if _, err := os.Stat(c.sessionPath()); !os.IsNotExist(err) {
		t.Error("session file should be deleted after logout")
	}
	t.Log("session file deleted: ok")
}

// --- Token refresh ---

func TestDoJSONRefreshOnUnauthorized(t *testing.T) {
	// Arrange: first /api/v1/notes call returns 401; /api/v1/auth/refresh returns 200;
	// second /api/v1/notes call returns 200.
	var notesHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("server: %s %s", r.Method, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/notes":
			if notesHits.Add(1) == 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token expired"})
			} else {
				writeJSON(w, http.StatusOK, map[string]any{"notes": []any{}, "total": 0})
			}
		case "/api/v1/auth/refresh":
			writeJSON(w, http.StatusOK, authResp("uid1", "u@example.com", "U"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.session = &Session{AccessToken: "expired-tok", RefreshToken: "refresh-tok", ServerURL: srv.URL}

	var result map[string]any
	status, err := c.DoJSON("GET", "/api/v1/notes", nil, &result)
	t.Logf("status=%d err=%v notesHits=%d", status, err, notesHits.Load())
	if err != nil {
		t.Fatalf("DoJSON after refresh: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected 200 after refresh, got %d", status)
	}
	if notesHits.Load() != 2 {
		t.Errorf("expected 2 hits on /api/v1/notes (initial + retry), got %d", notesHits.Load())
	}
	if c.session.AccessToken != "access-tok" {
		t.Errorf("expected token refreshed, got %q", c.session.AccessToken)
	}
}

func TestDoJSONRefreshFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("server: %s %s", r.Method, r.URL.Path)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.session = &Session{AccessToken: "bad-tok", RefreshToken: "bad-refresh", ServerURL: srv.URL}

	status, err := c.DoJSON("GET", "/api/v1/notes", nil, nil)
	t.Logf("status=%d err=%v", status, err)
	if status != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", status)
	}
}

func TestDoJSONNoRefreshWhenNoSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	// No session set — should not attempt refresh
	status, err := c.DoJSON("GET", "/api/v1/notes", nil, nil)
	t.Logf("no session: status=%d err=%v", status, err)
	if status != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", status)
	}
}

func TestDoJSONErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing field"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.session = &Session{AccessToken: "tok"}
	_, err := c.DoJSON("POST", "/api/v1/notes", map[string]string{}, nil)
	t.Logf("error body: %v", err)
	if err == nil {
		t.Error("expected error from 4xx response")
	}
	if err.Error() != "missing field" {
		t.Errorf("expected server error message, got %q", err.Error())
	}
}

// --- Config and session persistence ---

func TestConfigRoundtrip(t *testing.T) {
	c := newTestClient(t, nil)
	cfg := &Config{ServerURL: "http://notes.example.com", DeviceID: "my-laptop"}
	if err := c.saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	loaded, err := c.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	t.Logf("config: server_url=%s device_id=%s", loaded.ServerURL, loaded.DeviceID)
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("server_url: got %q, want %q", loaded.ServerURL, cfg.ServerURL)
	}
	if loaded.DeviceID != cfg.DeviceID {
		t.Errorf("device_id: got %q, want %q", loaded.DeviceID, cfg.DeviceID)
	}
}

func TestSessionRoundtrip(t *testing.T) {
	c := newTestClient(t, nil)
	c.session = &Session{
		AccessToken:  "acc",
		RefreshToken: "ref",
		UserID:       "uid",
		Email:        "e@example.com",
		DisplayName:  "Eve",
		ServerURL:    "http://srv",
	}
	if err := c.saveSession(); err != nil {
		t.Fatalf("saveSession: %v", err)
	}

	s, err := c.loadSession()
	if err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	t.Logf("session: user=%s email=%s access_token=%s", s.UserID, s.Email, s.AccessToken)
	if s.AccessToken != "acc" || s.RefreshToken != "ref" {
		t.Errorf("tokens: got access=%q refresh=%q", s.AccessToken, s.RefreshToken)
	}
	if s.Email != "e@example.com" {
		t.Errorf("email: got %q", s.Email)
	}
}

func TestSessionFilePermissions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, authResp("uid1", "u@example.com", "U"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Login(srv.URL, "u@example.com", "pass", "dev"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	info, err := os.Stat(c.sessionPath())
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	mode := info.Mode().Perm()
	t.Logf("session file mode: %04o", mode)
	if mode != 0600 {
		t.Errorf("session file should be 0600, got %04o", mode)
	}
}

func TestIsLoggedInFalseWithNoSession(t *testing.T) {
	c := newTestClient(t, nil)
	if c.IsLoggedIn() {
		t.Error("fresh client should not be logged in")
	}
	t.Log("IsLoggedIn=false: ok")
}

func TestNewWithDirLoadsExistingSession(t *testing.T) {
	dir := t.TempDir()

	// Write a session file directly
	sess := &Session{AccessToken: "existing-tok", RefreshToken: "r", ServerURL: "http://srv"}
	data, _ := json.MarshalIndent(sess, "", "  ")
	os.WriteFile(dir+"/session.json", data, 0600)

	c, err := NewWithDir(dir)
	if err != nil {
		t.Fatalf("NewWithDir: %v", err)
	}
	t.Logf("loaded session: access_token=%s server_url=%s", c.session.AccessToken, c.BaseURL)
	if !c.IsLoggedIn() {
		t.Error("expected client to be logged in from existing session")
	}
	if c.BaseURL != "http://srv" {
		t.Errorf("BaseURL from session: got %q", c.BaseURL)
	}
}
