package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Client struct {
	BaseURL    string
	DeviceID   string
	httpClient *http.Client
	configDir  string
	session    *Session
}

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	ServerURL    string `json:"server_url"`
}

// Config stored in ~/.notesd/config.toml
type Config struct {
	ServerURL string `toml:"server_url"`
	DeviceID  string `toml:"device_id"`
}

func New() (*Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	configDir := filepath.Join(home, ".notesd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		configDir:  configDir,
	}

	// Load config
	cfg, _ := c.loadConfig()
	if cfg != nil {
		c.BaseURL = cfg.ServerURL
		c.DeviceID = cfg.DeviceID
	}

	// Load session
	c.session, _ = c.loadSession()
	if c.session != nil && c.BaseURL == "" {
		c.BaseURL = c.session.ServerURL
	}

	// Default device ID to hostname
	if c.DeviceID == "" {
		c.DeviceID, _ = os.Hostname()
	}

	return c, nil
}

func (c *Client) IsLoggedIn() bool {
	return c.session != nil && c.session.AccessToken != ""
}

func (c *Client) SessionInfo() *Session {
	return c.session
}

func (c *Client) ConfigDir() string {
	return c.configDir
}

// DoJSON makes an authenticated JSON request and decodes the response.
// If the access token is expired (401), it attempts a refresh and retries.
func (c *Client) DoJSON(method, path string, body, result any) (int, error) {
	status, err := c.doJSONOnce(method, path, body, result)
	if status == http.StatusUnauthorized && c.session != nil && c.session.RefreshToken != "" {
		// Try refreshing the token
		if refreshErr := c.refreshTokens(); refreshErr == nil {
			return c.doJSONOnce(method, path, body, result)
		}
	}
	return status, err
}

func (c *Client) doJSONOnce(method, path string, body, result any) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.session != nil && c.session.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.session.AccessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if result != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	} else if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return resp.StatusCode, fmt.Errorf("%s", errResp.Error)
		}
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

// Auth types matching the server API

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

func (c *Client) Login(serverURL, email, password, deviceID string) error {
	c.BaseURL = serverURL
	c.DeviceID = deviceID

	var resp AuthResponse
	status, err := c.doJSONOnce("POST", "/api/v1/auth/login", map[string]string{
		"email":     email,
		"password":  password,
		"device_id": deviceID,
	}, &resp)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("login failed (HTTP %d)", status)
	}

	c.session = &Session{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		UserID:       resp.User.ID,
		Email:        resp.User.Email,
		DisplayName:  resp.User.DisplayName,
		ServerURL:    serverURL,
	}

	if err := c.saveSession(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	if err := c.saveConfig(&Config{ServerURL: serverURL, DeviceID: deviceID}); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}

func (c *Client) Register(serverURL, email, password, displayName string) error {
	c.BaseURL = serverURL

	status, err := c.doJSONOnce("POST", "/api/v1/auth/register", map[string]string{
		"email":        email,
		"password":     password,
		"display_name": displayName,
	}, nil)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("register failed (HTTP %d)", status)
	}
	return nil
}

func (c *Client) Logout() error {
	if c.session != nil && c.session.AccessToken != "" {
		c.DoJSON("POST", "/api/v1/auth/logout", nil, nil)
	}
	c.session = nil
	return c.deleteSession()
}

func (c *Client) refreshTokens() error {
	var resp AuthResponse
	status, err := c.doJSONOnce("POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": c.session.RefreshToken,
	}, &resp)
	if err != nil || status != http.StatusOK {
		return fmt.Errorf("refresh failed")
	}

	c.session.AccessToken = resp.AccessToken
	c.session.RefreshToken = resp.RefreshToken
	return c.saveSession()
}

// Session persistence

func (c *Client) sessionPath() string {
	return filepath.Join(c.configDir, "session.json")
}

func (c *Client) configPath() string {
	return filepath.Join(c.configDir, "config.toml")
}

func (c *Client) loadSession() (*Session, error) {
	data, err := os.ReadFile(c.sessionPath())
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) saveSession() error {
	data, err := json.MarshalIndent(c.session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.sessionPath(), data, 0600)
}

func (c *Client) deleteSession() error {
	return os.Remove(c.sessionPath())
}

func (c *Client) loadConfig() (*Config, error) {
	data, err := os.ReadFile(c.configPath())
	if err != nil {
		return nil, err
	}
	// Simple TOML parsing for two keys — avoids dependency
	cfg := &Config{}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := bytes.SplitN(line, []byte("="), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		// Strip quotes
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		switch key {
		case "server_url":
			cfg.ServerURL = val
		case "device_id":
			cfg.DeviceID = val
		}
	}
	return cfg, nil
}

func (c *Client) saveConfig(cfg *Config) error {
	content := fmt.Sprintf("server_url = %q\ndevice_id = %q\n", cfg.ServerURL, cfg.DeviceID)
	return os.WriteFile(c.configPath(), []byte(content), 0600)
}
