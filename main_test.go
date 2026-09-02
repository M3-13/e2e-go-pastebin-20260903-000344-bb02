package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	newApp(appConfig{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestUnknownPathReturns404JSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	newApp(appConfig{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("expected JSON error body, got %s", rec.Body.String())
	}
}

func TestMethodMismatchReturns405WithAllow(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/pastes", nil)
	newApp(appConfig{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow == "" {
		t.Fatal("expected Allow header on 405")
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("expected JSON error body, got %s", rec.Body.String())
	}
}

func TestHSTSOnlyInTLSMode(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	newApp(appConfig{}).ServeHTTP(rec, req)
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected no HSTS header without TLS, got %q", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	newApp(appConfig{certFile: "cert.pem", keyFile: "key.pem"}).ServeHTTP(rec, req)
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("expected HSTS header when TLS is enabled")
	}
}

func TestCORSAllowedOriginReflectedWithVary(t *testing.T) {
	cfg := appConfig{corsOrigins: []string{"https://example.com", "http://localhost:5173"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	req.Header.Set("Origin", "https://example.com")
	newApp(cfg).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected allowed origin reflected, got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary: Origin, got %q", got)
	}
}

func TestCORSDisallowedOriginNotReflected(t *testing.T) {
	cfg := appConfig{corsOrigins: []string{"https://example.com"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	req.Header.Set("Origin", "https://evil.com")
	newApp(cfg).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for disallowed origin, got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "" {
		t.Fatalf("expected no Vary header for disallowed origin, got %q", got)
	}
}

func TestCORSNoWildcardEverEmitted(t *testing.T) {
	cfg := appConfig{corsOrigins: []string{"https://example.com"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	req.Header.Set("Origin", "https://example.com")
	newApp(cfg).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatalf("wildcard '*' must never be emitted, got %q", got)
	}
}

func TestCORSEmptyAllowlistDenies(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	req.Header.Set("Origin", "https://example.com")
	newApp(appConfig{}).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected deny by default (no header), got %q", got)
	}
}

func TestParseOrigins(t *testing.T) {
	if got := parseOrigins(""); got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}
	if got := parseOrigins("  "); got != nil {
		t.Fatalf("expected nil for whitespace input, got %v", got)
	}

	got := parseOrigins("https://a.com, http://b.com ,,https://c.com,")
	want := []string{"https://a.com", "http://b.com", "https://c.com"}
	if len(got) != len(want) {
		t.Fatalf("expected %d origins, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("origin %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}
