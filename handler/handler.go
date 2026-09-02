package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"pastebin/store"
)

// Handler holds the dependencies shared by all paste handlers.
type Handler struct {
	store *store.Store
}

// NewHandler returns a Handler wired to the given store.
func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a uniform JSON error object with the given status code.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

type contextKey string

const pasteIDKey contextKey = "pasteID"

// WithPasteID returns a copy of r with the paste path ID stored in its context.
func WithPasteID(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), pasteIDKey, id)
	return r.WithContext(ctx)
}

// PasteID returns the paste path ID previously stored via WithPasteID, or the
// empty string if none is present.
func PasteID(r *http.Request) string {
	id, _ := r.Context().Value(pasteIDKey).(string)
	return id
}
