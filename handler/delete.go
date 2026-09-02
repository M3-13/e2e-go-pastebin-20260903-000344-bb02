package handler

import "net/http"

// DeletePaste handles DELETE /pastes/{id}.
func (h *Handler) DeletePaste(w http.ResponseWriter, r *http.Request) {
	id := PasteID(r)
	if !h.store.Delete(id) {
		WriteError(w, http.StatusNotFound, "paste not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
