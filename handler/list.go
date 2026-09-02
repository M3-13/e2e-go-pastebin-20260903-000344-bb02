package handler

import (
	"net/http"
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

// ListPastes handles GET /pastes. It returns the metadata of every non-expired
// paste as a JSON list, without the content field.
func (h *Handler) ListPastes(w http.ResponseWriter, r *http.Request) {
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
