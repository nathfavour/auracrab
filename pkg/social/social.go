package social

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/config"
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

type SocialConfig struct {
	Enabled      bool          `json:"enabled"`
	Platforms    []string      `json:"platforms"`
	PostInterval time.Duration `json:"post_interval"`
	Prompt       string        `json:"prompt"`
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

func LoadSocialConfig() (*SocialConfig, error) {
	path := filepath.Join(config.DataDir(), "social_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SocialConfig{
				Enabled:      false,
				Platforms:    []string{"threads"},
				PostInterval: 6 * time.Hour,
				Prompt:       "Write a developer joke or observation about the agentic future of software.",
			}, nil
		}
		return nil, err
	}
	var cfg SocialConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveSocialConfig(cfg *SocialConfig) error {
	path := filepath.Join(config.DataDir(), "social_config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (m *Manager) Start(ctx context.Context, querier ContextualQuerier) {
	log.Println("[Social] Starting social daemon loop...")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastPostTime time.Time

	for {
		select {
		case <-ctx.Done():
			log.Println("[Social] Stopping social daemon loop.")
			return
		case <-ticker.C:
			cfg, err := LoadSocialConfig()
			if err != nil {
				log.Printf("[Social] Error loading social config: %v", err)
				continue
			}

			if !cfg.Enabled {
				continue
			}

			if len(cfg.Platforms) == 0 {
				continue
			}

			interval := cfg.PostInterval
			if interval < 10*time.Second {
				interval = 6 * time.Hour
			}

			if time.Since(lastPostTime) >= interval {
				log.Println("[Social] Time to post. Querying AI model for content...")
				prompt := cfg.Prompt
				if prompt == "" {
					prompt = "Write a developer joke or observation about the agentic future of software."
				}

				resp, err := querier.QueryWithContext(ctx, prompt, "social_post_generation")
				if err != nil {
					log.Printf("[Social] Error generating social post: %v", err)
					continue
				}

				content := strings.TrimSpace(resp.Text)
				if content == "" {
					log.Println("[Social] Generated content was empty, skipping.")
					continue
				}

				log.Printf("[Social] Posting generated content to %v: %s", cfg.Platforms, content)
				results := m.PostToAll(ctx, content, cfg.Platforms)
				log.Printf("[Social] Posting results:\n%s", results)

				lastPostTime = time.Now()
			}
		}
	}
}
