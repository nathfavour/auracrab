package skills

import (
	"context"
	"encoding/json"
	"sync"
)

// Skill interface defines what a skill can do.
type Skill interface {
	Name() string
	Description() string
	Manifest() []byte
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry struct {
	skills map[string]Skill
	mu     sync.RWMutex
}

var (
	defaultRegistry *Registry
	once            sync.Once
)

func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = &Registry{
			skills: make(map[string]Skill),
		}
		// Auto-register built-in skills
		defaultRegistry.Register(&BrowserSkill{})
		defaultRegistry.Register(&BrowserAgentSkill{})
		defaultRegistry.Register(NewSocialSkill())
		defaultRegistry.Register(&AutoCommitSkill{})
		defaultRegistry.Register(&SystemSkill{})
	})
	return defaultRegistry
}

func (r *Registry) Register(s Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[s.Name()] = s
}

func (r *Registry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[name]
	return s, ok
}

func (r *Registry) List() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []Skill
	for _, s := range r.skills {
		list = append(list, s)
	}
	return list
}
