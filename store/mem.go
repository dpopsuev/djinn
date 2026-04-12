package store

import (
	"fmt"
	"sync"
)

var _ KeyStore = (*MemStore)(nil)

// MemStore is an in-memory KeyStore for testing. Thread-safe.
type MemStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemStore creates an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{data: make(map[string][]byte)}
}

func (s *MemStore) Put(key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.data[key] = cp
	return nil
}

func (s *MemStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	cp := make([]byte, len(d))
	copy(cp, d)
	return cp, nil
}

func (s *MemStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *MemStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	delete(s.data, key)
	return nil
}

func (s *MemStore) Close() error { return nil }

// Len returns the number of keys. Test helper.
func (s *MemStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
