package handler

import (
	"net/http"
	"os"
	"time"
)

// pasteMeta is the metadata of a paste as returned by GET /pastes, without the
// content field.
type pasteMeta struct {
	ID        string     `json:"id"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// apiKeyAuthorized reports whether the request carries a valid API key in the
// X-API-Key header. The expected key is read from PASTEBIN_API_KEY; an empty
// configured key means authorization is disabled and every request is rejected.
func apiKeyAuthorized(r *http.Request) bool {
	expected := os.Getenv("PASTEBIN_API_KEY")
	if expected == "" {
		return false
	}
	return r.Header.Get("X-API-Key") == expected
}

// ListPastes handles GET /pastes. It requires a valid X-API-Key header, then
// returns the metadata of every non-expired paste as a JSON list, without the
// content field.
func (h *Handler) ListPastes(w http.ResponseWriter, r *http.Request) {
	if !apiKeyAuthorized(r) {
		WriteError(w, http.StatusUnauthorized, "invalid or missing API key")
		return
	}

	pastes := h.store.List()

	out := make([]pasteMeta, 0, len(pastes))
	for _, p := range pastes {
		out = append(out, pasteMeta{
			ID:        p.ID,
			Language:  p.Language,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
		})
	}

	WriteJSON(w, http.StatusOK, out)
}
