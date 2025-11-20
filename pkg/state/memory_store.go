package state

import (
	"sync"
)

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string][]byte),
	}
}

func (s *MemoryStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if val, ok := s.data[string(key)]; ok {
		copyVal := make([]byte, len(val))
		copy(copyVal, val)
		return copyVal, nil
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) Set(key []byte, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyVal := make([]byte, len(value))
	copy(copyVal, value)
	s.data[string(key)] = copyVal
	return nil
}

func (s *MemoryStore) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[string(key)]; !ok {
		return ErrNotFound
	}
	delete(s.data, string(key))
	return nil
}
