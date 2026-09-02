package handler

import "net/http"

// CreatePaste handles POST /pastes.
func (h *Handler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "not implemented")
}
