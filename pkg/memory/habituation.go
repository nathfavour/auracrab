package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nathfavour/auracrab/pkg/config"
)

type HabitRecord struct {
	Goal  string   `json:"goal"`
	Steps []string `json:"steps"`
}

type HabitStore struct {
	mu     sync.RWMutex
	path   string
	habits map[string]HabitRecord
}

var (
	habitStore *HabitStore
	habitOnce  sync.Once
)

func GetHabitStore() *HabitStore {
	habitOnce.Do(func() {
		path := filepath.Join(config.DataDir(), "habituation.json")
		store := &HabitStore{
			path:   path,
			habits: make(map[string]HabitRecord),
		}
		store.load()
		habitStore = store
	})
	return habitStore
}

func (s *HabitStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = json.Unmarshal(data, &s.habits)
}

func (s *HabitStore) save() {
	s.mu.RLock()
	data, _ := json.MarshalIndent(s.habits, "", "  ")
	s.mu.RUnlock()
	_ = os.WriteFile(s.path, data, 0644)
}

// Learn records a successful sequence of steps for a given goal.
func (s *HabitStore) Learn(goal string, steps []string) {
	s.mu.Lock()
	// Normalize goal for better matching (simple lowercase)
	key := strings.ToLower(strings.TrimSpace(goal))
	s.habits[key] = HabitRecord{
		Goal:  goal,
		Steps: steps,
	}
	s.mu.Unlock()
	s.save()
}

// Recall attempts to find a cached plan for a similar goal.
func (s *HabitStore) Recall(goal string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(goal))
	
	// Direct match first
	if habit, ok := s.habits[key]; ok {
		return habit.Steps, true
	}

	// Fuzzy match: If the goal contains a known habit goal
	for k, v := range s.habits {
		if strings.Contains(key, k) || strings.Contains(k, key) {
			return v.Steps, true
		}
	}

	return nil, false
}
