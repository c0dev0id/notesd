package model

import (
	"crypto/rand"
	"fmt"
	"time"
)

// NewID generates a UUID v4 string.
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// NowMillis returns the current UTC time truncated to millisecond precision.
func NowMillis() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Note struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	Title            string     `json:"title"`
	Content          string     `json:"content"`
	Type             string     `json:"type"`
	ModifiedAt       time.Time  `json:"modified_at"`
	ModifiedByDevice string     `json:"modified_by_device"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type Todo struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	NoteID           *string    `json:"note_id,omitempty"`
	LineRef          *string    `json:"line_ref,omitempty"`
	Content          string     `json:"content"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	Completed        bool       `json:"completed"`
	ModifiedAt       time.Time  `json:"modified_at"`
	ModifiedByDevice string     `json:"modified_by_device"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// RefreshToken tracks issued refresh tokens for rotation and revocation.
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// API request types

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateNoteRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
}

type UpdateNoteRequest struct {
	Title    *string `json:"title"`
	Content  *string `json:"content"`
	Type     *string `json:"type"`
	DeviceID string  `json:"device_id"`
}

type CreateTodoRequest struct {
	NoteID   *string    `json:"note_id,omitempty"`
	LineRef  *string    `json:"line_ref,omitempty"`
	Content  string     `json:"content"`
	DueDate  *time.Time `json:"due_date,omitempty"`
	DeviceID string     `json:"device_id"`
}

type UpdateTodoRequest struct {
	Content   *string    `json:"content,omitempty"`
	DueDate   *time.Time `json:"due_date,omitempty"`
	Completed *bool      `json:"completed,omitempty"`
	NoteID    *string    `json:"note_id,omitempty"`
	LineRef   *string    `json:"line_ref,omitempty"`
	DeviceID  string     `json:"device_id"`
}

type SyncPushRequest struct {
	Notes    []Note `json:"notes"`
	Todos    []Todo `json:"todos"`
	DeviceID string `json:"device_id"`
}

// API response types

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type NoteListResponse struct {
	Notes  []Note `json:"notes"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type TodoListResponse struct {
	Todos  []Todo `json:"todos"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type SyncChangesResponse struct {
	Notes         []Note `json:"notes"`
	Todos         []Todo `json:"todos"`
	SyncTimestamp int64  `json:"sync_timestamp"`
}

type SyncPushResponse struct {
	Conflicts []SyncConflict `json:"conflicts,omitempty"`
	Accepted  int            `json:"accepted"`
	Timestamp int64          `json:"sync_timestamp"`
}

type SyncConflict struct {
	Type       string `json:"type"` // "note" or "todo"
	ID         string `json:"id"`
	ServerNote *Note  `json:"server_note,omitempty"`
	ServerTodo *Todo  `json:"server_todo,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
