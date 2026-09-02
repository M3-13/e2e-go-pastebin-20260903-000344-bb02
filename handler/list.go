package handler

import "net/http"

// ListPastes handles GET /pastes.
func (h *Handler) ListPastes(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "not implemented")
}
