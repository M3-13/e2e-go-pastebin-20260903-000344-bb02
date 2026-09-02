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

	p := model.Paste{
		Content:   req.Content,
		Language:  req.Language,
		CreatedAt: time.Now(),
	}

	if req.ExpiresInSeconds > 0 {
		expires := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
		p.ExpiresAt = &expires
	}

	for {
		id, err := newPasteID()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "could not generate id")
			return
		}
		p.ID = id
		if h.store.Create(p) {
			break
		}
	}

	WriteJSON(w, http.StatusCreated, map[string]string{"id": p.ID})
}

// newPasteID returns a 32-character lowercase hex string built from 16
// cryptographically random bytes (128 bits of entropy).
func newPasteID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
