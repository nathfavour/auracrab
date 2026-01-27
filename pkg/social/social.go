package social

import (
"context"
"fmt"
)

// Platform defines the interface for social media automation.
type Platform interface {
	Name() string
	Post(ctx context.Context, content string) (string, error)
	GetFeed(ctx context.Context, limit int) ([]Post, error)
}

type Post struct {
	ID      string
	Author  string
	Content string
	URL     string
}

type Manager struct {
	platforms map[string]Platform
}

func NewManager() *Manager {
	return &Manager{
		platforms: make(map[string]Platform),
	}
}

func (m *Manager) Register(p Platform) {
	m.platforms[p.Name()] = p
}

func (m *Manager) GetPlatform(name string) (Platform, error) {
	p, ok := m.platforms[name]
	if !ok {
		return nil, fmt.Errorf("platform %s not supported", name)
	}
	return p, nil
}

func (m *Manager) ListPlatforms() []string {
	var list []string
	for k := range m.platforms {
		list = append(list, k)
	}
	return list
}
