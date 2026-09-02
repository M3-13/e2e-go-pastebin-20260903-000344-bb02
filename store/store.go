package store

import (
	"sync"
	"time"

	"pastebin/model"
)

// Store is a thread-safe in-memory store of pastes.
type Store struct {
	mu     sync.Mutex
	pastes map[string]model.Paste
}

// New returns an empty, ready-to-use Store.
func New() *Store {
	return &Store{
		pastes: make(map[string]model.Paste),
	}
}

// Create stores p and reports success. It returns false if a paste with the
// same ID already exists.
func (s *Store) Create(p model.Paste) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pastes[p.ID]; ok {
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
