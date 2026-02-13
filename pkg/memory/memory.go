package memory

import (
"encoding/json"
"os"
"path/filepath"
"sync"

"github.com/nathfavour/auracrab/pkg/config"
)

// Store is a simple persistent key-value store.
type Store struct {
	data map[string]interface{}
	path string
	mu   sync.RWMutex
}

func NewStore(name string) (*Store, error) {
	dir := filepath.Join(config.DataDir(), "memory")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, name+".json")
	s := &Store{
		data: make(map[string]interface{}),
		path: path,
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

func (s *Store) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return s.save()
}

func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return s.save()
}

type Fact struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	MissionID   string `json:"mission_id,omitempty"`
}

func (s *Store) SaveFact(key, value, desc, missionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	facts, ok := s.data["facts"].([]interface{})
	if !ok {
		facts = []interface{}{}
	}

	newFact := Fact{key, value, desc, missionID}
	facts = append(facts, newFact)
	s.data["facts"] = facts
	return s.save()
}

func (s *Store) ListFacts() []Fact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Fact
	if facts, ok := s.data["facts"]; ok {
		data, _ := json.Marshal(facts)
		_ = json.Unmarshal(data, &result)
	}
	return result
}

func (s *Store) load() error {
	f, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(f, &s.data)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
