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
