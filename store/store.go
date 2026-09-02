package store

import (
	"sync"
	"time"

	"pastebin/model"
)

// MaxPastes is the maximum number of pastes the store may hold. Once the map
// reaches this size, Create reports failure even for a brand-new ID.
var MaxPastes int = 10000

// CleanupInterval is how often the background goroutine started by New purges
// expired pastes. It is a variable so tests can shorten it.
var CleanupInterval = time.Minute

// Store is a thread-safe in-memory store of pastes.
type Store struct {
	mu     sync.Mutex
	pastes map[string]model.Paste
}

// New returns an empty, ready-to-use Store and starts a background goroutine
// that periodically removes expired pastes.
func New() *Store {
	s := &Store{
		pastes: make(map[string]model.Paste),
	}
	go s.cleanupLoop(CleanupInterval)
	return s
}

// cleanupLoop periodically removes expired pastes until the ticker is stopped.
// It runs for the lifetime of the process; the interval is fixed at start-up.
func (s *Store) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.removeExpired()
	}
}

// removeExpired deletes every paste whose ExpiresAt is in the past. It is used
// by the background cleanup loop; Get and List keep their own lazy removal.
func (s *Store) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, p := range s.pastes {
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			delete(s.pastes, id)
		}
	}
}

// Create stores p and reports success. It returns false if a paste with the
// same ID already exists, or if the store already holds MaxPastes entries.
func (s *Store) Create(p model.Paste) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pastes[p.ID]; ok {
		return false
	}
	if len(s.pastes) >= MaxPastes {
		return false
	}
	s.pastes[p.ID] = p
	return true
}

// Get returns the paste with the given id. Expired pastes are removed from the
// store on access and reported as not found.
func (s *Store) Get(id string) (model.Paste, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pastes[id]
	if !ok {
		return model.Paste{}, false
	}
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		delete(s.pastes, id)
		return model.Paste{}, false
	}
	return p, true
}

// List returns all non-expired pastes, removing expired entries from the store.
func (s *Store) List() []model.Paste {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, p := range s.pastes {
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			delete(s.pastes, id)
		}
	}
	out := make([]model.Paste, 0, len(s.pastes))
	for _, p := range s.pastes {
		out = append(out, p)
	}
	return out
}

// Delete removes the paste with the given id and reports success. It returns
// false if no such paste exists.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pastes[id]; !ok {
		return false
	}
	delete(s.pastes, id)
	return true
}
