package memory

import (
	"fmt"
	"sync"
	"time"
)

type EphemeralStore interface {
	Set(key string, value string, expiration time.Duration) error
	Get(key string) (string, error)
}

type SimpleEphemeralStore struct {
	data map[string]entry
	mu   sync.RWMutex
}

type entry struct {
	value      string
	expiration time.Time
}

func NewSimpleEphemeralStore() *SimpleEphemeralStore {
	return &SimpleEphemeralStore{
		data: make(map[string]entry),
	}
}

func (s *SimpleEphemeralStore) Set(key string, value string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = entry{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	return nil
}

func (s *SimpleEphemeralStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok || time.Now().After(e.expiration) {
		return "", fmt.Errorf("key not found or expired")
	}
	return e.value, nil
}
