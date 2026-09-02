package handler

import "net/http"

// GetPaste handles GET /pastes/{id}.
func (h *Handler) GetPaste(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "not implemented")
}
