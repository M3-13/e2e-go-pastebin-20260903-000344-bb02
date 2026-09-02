package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pastebin/model"
	"pastebin/store"
)

func newGetHandler() *Handler {
	return NewHandler(store.New())
}

func doGet(h *Handler, id string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pastes/"+id, nil)
	req = WithPasteID(req, id)
	h.GetPaste(rec, req)
	return rec
}

func TestGetPasteExisting(t *testing.T) {
	h := newGetHandler()
	p := model.Paste{
		ID:        "aabbccddeeff00112233445566778899",
		Content:   "hello world",
		Language:  "text",
		CreatedAt: time.Now(),
	}
	h.store.Create(p)

	rec := doGet(h, p.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got model.Paste
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected valid JSON body, got error: %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("expected id %q, got %q", p.ID, got.ID)
	}
	if got.Content != p.Content {
		t.Fatalf("expected content %q, got %q", p.Content, got.Content)
	}
	if got.Language != p.Language {
		t.Fatalf("expected language %q, got %q", p.Language, got.Language)
	}
}

func TestGetPasteUnknownID(t *testing.T) {
	h := newGetHandler()

	rec := doGet(h, "deadbeefdeadbeefdeadbeefdeadbeef")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !jsonErrorBody(rec) {
		t.Fatalf("expected JSON error body, got %s", rec.Body.String())
	}
}

func TestGetPasteExpired(t *testing.T) {
	h := newGetHandler()
	expired := time.Now().Add(-time.Second)
	p := model.Paste{
		ID:        "aabbccddeeff00112233445566778899",
		Content:   "expired content",
		Language:  "text",
		CreatedAt: time.Now().Add(-time.Minute),
		ExpiresAt: &expired,
	}
	h.store.Create(p)

	rec := doGet(h, p.ID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for expired paste, got %d", rec.Code)
	}
	if !jsonErrorBody(rec) {
		t.Fatalf("expected JSON error body, got %s", rec.Body.String())
	}
}

func TestGetPasteNoExpiry(t *testing.T) {
	h := newGetHandler()
	p := model.Paste{
		ID:        "aabbccddeeff00112233445566778899",
		Content:   "permanent",
		Language:  "text",
		CreatedAt: time.Now(),
	}
	h.store.Create(p)

	rec := doGet(h, p.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for paste without expiry, got %d", rec.Code)
	}
	var got model.Paste
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected valid JSON body, got error: %v", err)
	}
	if got.Content != p.Content {
		t.Fatalf("expected content %q, got %q", p.Content, got.Content)
	}
	if got.ExpiresAt != nil {
		t.Fatalf("expected no expires_at, got %v", got.ExpiresAt)
	}
}

func jsonErrorBody(rec *httptest.ResponseRecorder) bool {
	var m map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		return false
	}
	_, ok := m["error"]
	return ok
}
