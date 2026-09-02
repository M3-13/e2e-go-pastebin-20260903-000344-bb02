package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pastebin/model"
	"pastebin/store"
)

func newDeleteTestHandler() (*Handler, *store.Store) {
	s := store.New()
	h := NewHandler(s)
	return h, s
}

func deleteRequest(id string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+id, nil)
	return WithPasteID(req, id)
}

func TestDeletePasteExistingReturns204(t *testing.T) {
	h, s := newDeleteTestHandler()
	p := model.Paste{
		ID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Content:   "hello",
		Language:  "text",
		CreatedAt: time.Now(),
	}
	if !s.Create(p) {
		t.Fatal("failed to seed paste")
	}

	rec := httptest.NewRecorder()
	h.DeletePaste(rec, deleteRequest(p.ID))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("expected empty body, got %q", body)
	}

	if _, ok := s.Get(p.ID); ok {
		t.Fatal("expected paste to be gone after delete")
	}
	for _, listed := range s.List() {
		if listed.ID == p.ID {
			t.Fatal("deleted paste should not appear in List")
		}
	}
}

func TestDeletePasteUnknownReturns404(t *testing.T) {
	h, _ := newDeleteTestHandler()

	rec := httptest.NewRecorder()
	h.DeletePaste(rec, deleteRequest("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected error body")
	}
}
