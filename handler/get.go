package handler

import "net/http"

// GetPaste handles GET /pastes/{id}.
func (h *Handler) GetPaste(w http.ResponseWriter, r *http.Request) {
	id := PasteID(r)
	p, ok := h.store.Get(id)
	if !ok {
		WriteError(w, http.StatusNotFound, "paste not found")
		return
	}
	WriteJSON(w, http.StatusOK, p)
}
