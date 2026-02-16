package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Agent represents a unique agentic identity within the framework.
type Agent struct {
	ID          string    `json:"id"`
	Handle      string    `json:"handle"` // e.g., "auracrab", "assistant_1"
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	IsDefault   bool      `json:"is_default"`
}

type Registry struct {
	Agents map[string]*Agent `json:"agents"` // Map of Handle -> Agent
	path   string
	mu     sync.RWMutex
}

func NewRegistry() (*Registry, error) {
	path := filepath.Join(os.Getenv("HOME"), ".auracrab", "registry.json")
	r := &Registry{
		Agents: make(map[string]*Agent),
		path:   path,
	}

	if err := r.load(); err != nil {
		if os.IsNotExist(err) {
			// Initialize with default auracrab agent if registry doesn't exist
			defaultAgent := &Agent{
				ID:          "default-auracrab-id", // Will be replaced by UUID if we want, but keeping it stable for now
				Handle:      "auracrab",
				DisplayName: "Auracrab (Default)",
				CreatedAt:   time.Now(),
				IsDefault:   true,
			}
			r.Agents[defaultAgent.Handle] = defaultAgent
			_ = r.save()
		} else {
			return nil, err
		}
	}

	return r, nil
}

func (r *Registry) Create(handle, displayName string) (*Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Agents[handle]; exists {
		return nil, fmt.Errorf("agent with handle '%s' already exists", handle)
	}

	agent := &Agent{
		ID:          uuid.New().String(),
		Handle:      handle,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
		IsDefault:   false,
	}

	r.Agents[handle] = agent
	if err := r.save(); err != nil {
		return nil, err
	}

	return agent, nil
}

func (r *Registry) Get(handle string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.Agents[handle]
	return a, ok
}

func (r *Registry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*Agent
	for _, a := range r.Agents {
		list = append(list, a)
	}
	return list
}

func (r *Registry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, r)
}

func (r *Registry) save() error {
	_ = os.MkdirAll(filepath.Dir(r.path), 0755)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

// GetAgentDataDir returns the namespaced data directory for an agent.
func GetAgentDataDir(agentHandle string) string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".auracrab", "agents", agentHandle)
	_ = os.MkdirAll(path, 0755)
	return path
}
