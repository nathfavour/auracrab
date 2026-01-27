package skills

import (
"context"
"fmt"
"sync"
)

// Skill interface defines what a skill can do.
type Skill interface {
	Name() string
	Description() string
	Execute(ctx context.Context, action string, args map[string]interface{}) (string, error)
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

func (r *Registry) Execute(ctx context.Context, name string, action string, args map[string]interface{}) (string, error) {
	s, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("skill '%s' not found", name)
	}
	return s.Execute(ctx, action, args)
}
