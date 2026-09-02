package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pastebin/model"
	"pastebin/store"
)

const testAPIKey = "test-api-key"

func listRequest(apiKey string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	return req
}

func TestListPastes(t *testing.T) {
	t.Setenv("PASTEBIN_API_KEY", testAPIKey)

	s := store.New()
	h := NewHandler(s)

	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	p1 := model.Paste{
		ID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Content:   "hello",
		Language:  "go",
		CreatedAt: created,
	}
	p2 := model.Paste{
		ID:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Content:   "world",
		Language:  "python",
		CreatedAt: created,
	}
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	p2.ExpiresAt = &exp

	// An already-expired paste must not appear.
	p3 := model.Paste{
		ID:        "cccccccccccccccccccccccccccccccc",
		Content:   "expired",
		Language:  "text",
		CreatedAt: created,
	}
	expPast := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	p3.ExpiresAt = &expPast

	s.Create(p1)
	s.Create(p2)
	s.Create(p3)

	rec := httptest.NewRecorder()
	h.ListPastes(rec, listRequest(testAPIKey))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 pastes, got %d: %v", len(list), list)
	}

	for _, item := range list {
		if _, ok := item["content"]; ok {
			t.Errorf("metadata must not include content field, got %v", item)
		}
		if _, ok := item["id"]; !ok {
			t.Errorf("metadata must include id, got %v", item)
		}
		if _, ok := item["language"]; !ok {
			t.Errorf("metadata must include language, got %v", item)
		}
		if _, ok := item["created_at"]; !ok {
			t.Errorf("metadata must include created_at, got %v", item)
		}
		if _, ok := item["delete_token"]; ok {
			t.Errorf("metadata must not include delete_token field, got %v", item)
		}
	}

	ids := map[string]bool{}
	for _, item := range list {
		ids[item["id"].(string)] = true
	}
	if !ids[p1.ID] || !ids[p2.ID] {
		t.Errorf("expected ids of both non-expired pastes, got %v", ids)
	}
	if ids[p3.ID] {
		t.Errorf("expired paste must not appear, got %v", ids)
	}
}

func TestListPastesEmptyStore(t *testing.T) {
	t.Setenv("PASTEBIN_API_KEY", testAPIKey)

	s := store.New()
	h := NewHandler(s)

	rec := httptest.NewRecorder()
	h.ListPastes(rec, listRequest(testAPIKey))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Fatalf("expected empty JSON list [], got %q", body)
	}
}

func TestListPastesMissingKeyReturns401(t *testing.T) {
	t.Setenv("PASTEBIN_API_KEY", testAPIKey)

	s := store.New()
	h := NewHandler(s)

	rec := httptest.NewRecorder()
	h.ListPastes(rec, listRequest(""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected error body")
	}
}

func TestListPastesWrongKeyReturns401(t *testing.T) {
	t.Setenv("PASTEBIN_API_KEY", testAPIKey)

	s := store.New()
	h := NewHandler(s)

	rec := httptest.NewRecorder()
	h.ListPastes(rec, listRequest("wrong-key"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected error body")
	}
}

func TestListPastesEmptyEnvKeyReturns401(t *testing.T) {
	t.Setenv("PASTEBIN_API_KEY", "")

	s := store.New()
	h := NewHandler(s)

	rec := httptest.NewRecorder()
	h.ListPastes(rec, listRequest(testAPIKey))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
