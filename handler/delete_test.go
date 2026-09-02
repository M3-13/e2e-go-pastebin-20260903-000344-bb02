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

func deleteRequest(id, token string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+id, nil)
	if token != "" {
		req.Header.Set("X-Delete-Token", token)
	}
	return WithPasteID(req, id)
}

func seedPaste(s *store.Store, id, token string) {
	p := model.Paste{
		ID:          id,
		Content:     "hello",
		Language:    "text",
		CreatedAt:   time.Now(),
		DeleteToken: token,
	}
	if !s.Create(p) {
		panic("failed to seed paste")
	}
}

func TestDeletePasteWithValidTokenReturns204(t *testing.T) {
	h, s := newDeleteTestHandler()
	const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const token = "0123456789abcdef0123456789abcdef"
	seedPaste(s, id, token)

	rec := httptest.NewRecorder()
	h.DeletePaste(rec, deleteRequest(id, token))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("expected empty body, got %q", body)
	}

	if _, ok := s.Get(id); ok {
		t.Fatal("expected paste to be gone after delete")
	}
	for _, listed := range s.List() {
		if listed.ID == id {
			t.Fatal("deleted paste should not appear in List")
		}
	}
}

func TestDeletePasteMissingTokenReturns401(t *testing.T) {
	h, s := newDeleteTestHandler()
	const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	seedPaste(s, id, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	h.DeletePaste(rec, deleteRequest(id, ""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected error body")
	}

	if _, ok := s.Get(id); !ok {
		t.Fatal("paste must not be deleted on missing token")
	}
}

func TestDeletePasteWrongTokenReturns401(t *testing.T) {
	h, s := newDeleteTestHandler()
	const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	seedPaste(s, id, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	h.DeletePaste(rec, deleteRequest(id, "ffffffffffffffffffffffffffffffff"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected error body")
	}

	if _, ok := s.Get(id); !ok {
		t.Fatal("paste must not be deleted on wrong token")
	}
}

func TestDeletePasteUnknownWithTokenReturns404(t *testing.T) {
	h, _ := newDeleteTestHandler()

	rec := httptest.NewRecorder()
	h.DeletePaste(rec, deleteRequest("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "0123456789abcdef0123456789abcdef"))

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
