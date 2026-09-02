package model

import "time"

// Paste is a single paste stored by the API.
type Paste struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// DeleteToken authorizes deletion of this paste. It is never serialized
	// into GET/List responses.
	DeleteToken string `json:"-"`
}
