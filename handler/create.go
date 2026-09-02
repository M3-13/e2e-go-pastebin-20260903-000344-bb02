package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"pastebin/model"
)

const maxPasteBodyBytes = 1 << 20 // 1 MB

// maxExpireSeconds caps expires_in_seconds at roughly 10 years. It is checked
// before the value is converted to a time.Duration so that an unreasonably
// large request is rejected cleanly instead of overflowing the duration.
const maxExpireSeconds = 10 * 365 * 24 * 3600

// maxIDRetries bounds how many times CreatePaste regenerates a paste ID after a
// collision before giving up. Once the store has reached store.MaxPastes,
// store.Create fails fast and these retries exhaust immediately, yielding 503.
const maxIDRetries = 10

type createRequest struct {
	Content          string `json:"content"`
	Language         string `json:"language"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// CreatePaste handles POST /pastes.
func (h *Handler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPasteBodyBytes)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Content == "" {
		WriteError(w, http.StatusBadRequest, "content is required")
		return
	}

	if req.ExpiresInSeconds < 0 {
		WriteError(w, http.StatusBadRequest, "expires_in_seconds must be positive")
		return
	}
	if req.ExpiresInSeconds > maxExpireSeconds {
		WriteError(w, http.StatusBadRequest, "expires_in_seconds exceeds the maximum allowed")
		return
	}

	p := model.Paste{
		Content:   req.Content,
		Language:  req.Language,
		CreatedAt: time.Now(),
	}

	if req.ExpiresInSeconds > 0 {
		expires := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
		p.ExpiresAt = &expires
	}

	deleteToken, err := newSecret()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "could not generate delete token")
		return
	}
	p.DeleteToken = deleteToken

	for attempt := 0; attempt < maxIDRetries; attempt++ {
		id, err := newPasteID()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "could not generate id")
			return
		}
		p.ID = id
		if h.store.Create(p) {
			WriteJSON(w, http.StatusCreated, map[string]string{
				"id":           p.ID,
				"delete_token": p.DeleteToken,
			})
			return
		}
		// Create reported false: either an ID collision or the store has reached
		// store.MaxPastes. Retry on a fresh ID for the bounded number of attempts,
		// then fail as unavailable.
	}

	WriteError(w, http.StatusServiceUnavailable, "paste store is full")
}

// newPasteID returns a 32-character lowercase hex string built from 16
// cryptographically random bytes (128 bits of entropy).
func newPasteID() (string, error) {
	return newSecret()
}

// newSecret returns a 32-character lowercase hex string built from 16
// cryptographically random bytes (128 bits of entropy). It is used for both the
// paste ID and the delete token.
func newSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
