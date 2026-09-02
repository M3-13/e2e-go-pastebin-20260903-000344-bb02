package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pastebin/store"
)

func newTestServer() (*httptest.Server, *store.Store) {
	s := store.New()
	h := NewHandler(s)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreatePaste(w, r)
			return
		}
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}))
	return ts, s
}

func TestCreatePasteReturnsID(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(`{"content":"Hallo"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	id, ok := body["id"]
	if !ok || id == "" {
		t.Fatalf("expected non-empty id, got %q", id)
	}
	if len(id) != 32 {
		t.Fatalf("expected 32-char hex id, got %d chars: %q", len(id), id)
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("id contains non-hex character: %q", id)
		}
	}

	deleteToken, ok := body["delete_token"]
	if !ok || deleteToken == "" {
		t.Fatalf("expected non-empty delete_token, got %q", deleteToken)
	}
	if len(deleteToken) != 32 {
		t.Fatalf("expected 32-char hex delete_token, got %d chars: %q", len(deleteToken), deleteToken)
	}
}

func TestCreatePasteEmptyContent(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	for name, body := range map[string]string{
		"empty":   `{"content":""}`,
		"missing": `{"language":"go"}`,
	} {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatalf("POST failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestCreatePasteInvalidJSON(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(`{not valid json`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePasteBodyTooLarge(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	big := strings.Repeat("a", maxPasteBodyBytes+1)
	body := `{"content":"` + big + `"}`

	resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp.StatusCode)
	}
}

func TestCreatePasteExpiresAt(t *testing.T) {
	ts, s := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(`{"content":"Hallo","expires_in_seconds":3600}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	p, ok := s.Get(body["id"])
	if !ok {
		t.Fatalf("paste not found in store")
	}
	if p.ExpiresAt == nil {
		t.Fatalf("expected ExpiresAt to be set")
	}

	want := time.Now().Add(3600 * time.Second)
	diff := want.Sub(*p.ExpiresAt)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Fatalf("ExpiresAt not ~now+3600s: got %v, want %v", *p.ExpiresAt, want)
	}
}

func TestCreatePasteNoExpiry(t *testing.T) {
	ts, s := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(`{"content":"Hallo","expires_in_seconds":0}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	p, ok := s.Get(body["id"])
	if !ok {
		t.Fatalf("paste not found in store")
	}
	if p.ExpiresAt != nil {
		t.Fatalf("expected nil ExpiresAt, got %v", *p.ExpiresAt)
	}
}

func TestCreatePasteExpiresOverflow(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/pastes", "application/json", strings.NewReader(`{"content":"Hallo","expires_in_seconds":3153600001}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for overflow, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("expected JSON error body, got %v", body)
	}
}
