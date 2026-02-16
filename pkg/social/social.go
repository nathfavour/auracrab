package social

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

var (
	managerInstance *Manager
	once            sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		managerInstance = &Manager{
			platforms: make(map[string]Platform),
		}
		// Register default drivers
		managerInstance.Register(&XDriver{})
		managerInstance.Register(&LinkedInDriver{})
		managerInstance.Register(&FacebookDriver{})
		managerInstance.Register(&InstagramDriver{})
		managerInstance.Register(&ThreadsDriver{})
	})
	return managerInstance
}

func (m *Manager) Register(p Platform) {
	m.platforms[p.Name()] = p
}

func (m *Manager) PostToAll(ctx context.Context, content string, platforms []string) string {
	var results []string
	for _, name := range platforms {
		p, ok := m.platforms[name]
		if !ok {
			results = append(results, fmt.Sprintf("%s: [Error] Platform not found", name))
			continue
		}
		res, err := p.Post(ctx, content)
		if err != nil {
			results = append(results, fmt.Sprintf("%s: [Error] %v", name, err))
		} else {
			results = append(results, fmt.Sprintf("%s: [Success] %s", name, res))
		}
	}
	return strings.Join(results, "\n")
}

func (m *Manager) ListPlatforms() []string {
	var list []string
	for k := range m.platforms {
		list = append(list, k)
	}
	return list
}
