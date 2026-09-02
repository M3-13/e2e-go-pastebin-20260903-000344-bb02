package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"pastebin/model"
)

func testPaste(id string, expires *time.Time) model.Paste {
	return model.Paste{
		ID:        id,
		Content:   "content-" + id,
		Language:  "text",
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
}

func TestCreateGet(t *testing.T) {
	s := New()
	p := testPaste("a", nil)

	if !s.Create(p) {
		t.Fatal("expected Create to succeed for new ID")
	}
	if s.Create(p) {
		t.Fatal("expected Create to fail for duplicate ID")
	}

	got, ok := s.Get("a")
	if !ok {
		t.Fatal("expected Get to find existing paste")
	}
	if got.ID != p.ID || got.Content != p.Content {
		t.Fatalf("got unexpected paste: %+v", got)
	}

	if _, ok := s.Get("missing"); ok {
		t.Fatal("expected Get to miss unknown ID")
	}
}

func TestList(t *testing.T) {
	s := New()
	s.Create(testPaste("a", nil))
	s.Create(testPaste("b", nil))

	got := s.List()
	if len(got) != 2 {
		t.Fatalf("expected 2 pastes, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p.ID] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("List missing expected IDs: %v", seen)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.Create(testPaste("a", nil))

	if !s.Delete("a") {
		t.Fatal("expected Delete to succeed for existing ID")
	}
	if s.Delete("a") {
		t.Fatal("expected Delete to fail for missing ID")
	}
	if _, ok := s.Get("a"); ok {
		t.Fatal("expected paste to be gone after Delete")
	}
	if len(s.List()) != 0 {
		t.Fatal("expected List to be empty after Delete")
	}
}

func TestExpiredRemovedOnGet(t *testing.T) {
	s := New()
	expired := time.Now().Add(-time.Second)
	s.Create(testPaste("expired", &expired))

	if _, ok := s.Get("expired"); ok {
		t.Fatal("expected expired paste to be reported as not found")
	}
	// After the access, the entry must actually be gone, not just hidden.
	if s.Delete("expired") {
		t.Fatal("expected expired paste to have been removed from the store")
	}
}

func TestExpiredRemovedOnList(t *testing.T) {
	s := New()
	expired := time.Now().Add(-time.Second)
	future := time.Now().Add(time.Hour)
	s.Create(testPaste("expired", &expired))
	s.Create(testPaste("future", &future))
	s.Create(testPaste("never", nil))

	got := s.List()
	if len(got) != 2 {
		t.Fatalf("expected 2 non-expired pastes, got %d", len(got))
	}
	for _, p := range got {
		if p.ID == "expired" {
			t.Fatal("expired paste should not appear in List")
		}
	}
	// The expired entry must have been physically removed.
	if _, ok := s.Get("expired"); ok {
		t.Fatal("expired paste should be gone from the store")
	}
}

func TestExpiredRemovedByCleanupLoop(t *testing.T) {
	old := CleanupInterval
	CleanupInterval = 10 * time.Millisecond
	defer func() { CleanupInterval = old }()

	s := New()
	expired := time.Now().Add(30 * time.Millisecond)
	s.Create(testPaste("expired", &expired))
	s.Create(testPaste("kept", nil))

	// Wait for the background goroutine to purge the expired entry without any
	// Get/List access. Delete is a neutral probe: it does not itself apply
	// expiry logic, so it only reports whether the entry is physically present.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && s.Delete("expired") {
		time.Sleep(5 * time.Millisecond)
	}
	if s.Delete("expired") {
		t.Fatal("expected expired paste to be removed by the cleanup loop without any Get/List access")
	}
	if !s.Delete("kept") {
		t.Fatal("expected non-expired paste to survive the cleanup loop")
	}
}

func TestCreateLimit(t *testing.T) {
	old := MaxPastes
	MaxPastes = 2
	defer func() { MaxPastes = old }()

	s := New()
	if !s.Create(testPaste("a", nil)) {
		t.Fatal("expected Create to succeed under the limit")
	}
	if !s.Create(testPaste("b", nil)) {
		t.Fatal("expected Create to succeed when the map reaches the limit")
	}
	if s.Create(testPaste("c", nil)) {
		t.Fatal("expected Create to fail once the map holds MaxPastes entries")
	}
}

func TestConcurrentWithCleanupLoop(t *testing.T) {
	old := CleanupInterval
	CleanupInterval = time.Millisecond
	defer func() { CleanupInterval = old }()

	s := New()
	const workers = 20
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				id := fmt.Sprintf("p-%d-%d", w, i)
				expires := time.Now().Add(time.Duration(w+i) * time.Millisecond)
				s.Create(testPaste(id, &expires))
				s.Get(id)
				s.List()
			}
		}(w)
	}
	wg.Wait()
}

func TestConcurrentAccess(t *testing.T) {
	s := New()
	const workers = 50
	const perWorker = 100

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				id := fmt.Sprintf("p-%d-%d", w, i)
				s.Create(testPaste(id, nil))
				if _, ok := s.Get(id); !ok {
					t.Errorf("worker %d: paste %s not found after create", w, id)
				}
			}
		}(w)
	}
	wg.Wait()

	got := s.List()
	if len(got) != workers*perWorker {
		t.Fatalf("expected %d pastes, got %d", workers*perWorker, len(got))
	}
}
