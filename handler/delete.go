package handler

import "net/http"

// DeletePaste handles DELETE /pastes/{id}.
func (h *Handler) DeletePaste(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "not implemented")
}
