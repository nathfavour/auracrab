package crabs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Crab represents a user-defined specialized agent.
type Crab struct {
	ID           string   "json:\"id\""
	Name         string   "json:\"name\""
	Description  string   "json:\"description\""
	Instructions string   "json:\"instructions\""
	Skills       []string "json:\"skills\"" // Names of skills this crab can use
}

type Registry struct {
	dataDir string
	crabs   map[string]Crab
	mu      sync.RWMutex
}

func NewRegistry() (*Registry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(home, ".local", "share", "auracrab", "crabs")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	r := &Registry{
		dataDir: dataDir,
		crabs:   make(map[string]Crab),
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Register(c Crab) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.crabs[c.ID] = c
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(r.dataDir, c.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (r *Registry) List() ([]Crab, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []Crab
	for _, c := range r.crabs {
		list = append(list, c)
	}
	return list, nil
}

func (r *Registry) Get(id string) (Crab, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.crabs[id]
	if !ok {
		return Crab{}, fmt.Errorf("crab with ID '%s' not found", id)
	}
	return c, nil
}

func (r *Registry) load() error {
	files, err := filepath.Glob(filepath.Join(r.dataDir, "*.json"))
	if err != nil {
		return err
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var c Crab
		if err := json.Unmarshal(data, &c); err == nil {
			r.crabs[c.ID] = c
		}
	}
	return nil
}
