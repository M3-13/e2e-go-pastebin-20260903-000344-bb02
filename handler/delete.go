package handler

import (
	"crypto/subtle"
	"net/http"
)

// DeletePaste handles DELETE /pastes/{id}. The caller must present the paste's
// delete token in the X-Delete-Token header; without a valid token the request
// is rejected with 401.
func (h *Handler) DeletePaste(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Delete-Token")
	if token == "" {
		WriteError(w, http.StatusUnauthorized, "delete token required")
		return
	}

	id := PasteID(r)
	p, ok := h.store.Get(id)
	if !ok {
		WriteError(w, http.StatusNotFound, "paste not found")
		return
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(p.DeleteToken)) != 1 {
		WriteError(w, http.StatusUnauthorized, "invalid delete token")
		return
	}

	h.store.Delete(id)
	w.WriteHeader(http.StatusNoContent)
}
