package mission

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/config"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusAbandoned Status = "abandoned"
)

type Mission struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Goal        string    `json:"goal"`
	Deadline    time.Time `json:"deadline"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Temporal Awareness metrics
	EstimatedTTC time.Duration `json:"estimated_ttc"` // Time To Complete
	Progress     float64       `json:"progress"`      // 0.0 to 1.0
}

type Manager struct {
	missions map[string]*Mission
	mu       sync.RWMutex
	path     string
}

func NewManager() (*Manager, error) {
	path := filepath.Join(config.DataDir(), "missions.json")
	m := &Manager{
		missions: make(map[string]*Mission),
		path:     path,
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) CreateMission(title, desc, goal string, deadline time.Time) *Mission {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := time.Now().Format("20060102-150405")
	mission := &Mission{
		ID:          id,
		Title:       title,
		Description: desc,
		Goal:        goal,
		Deadline:    deadline,
		Status:      StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.missions[id] = mission
	m.save()
	return mission
}

func (m *Manager) GetActiveMission() *Mission {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, mission := range m.missions {
		if mission.Status == StatusActive {
			return mission
		}
	}
	return nil
}

func (m *Manager) UpdateProgress(id string, progress float64, ttc time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mission, ok := m.missions[id]
	if !ok {
		return os.ErrNotExist
	}

	mission.Progress = progress
	mission.EstimatedTTC = ttc
	mission.UpdatedAt = time.Now()
	m.save()
	return nil
}

func (m *Manager) TimeRemaining(id string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mission, ok := m.missions[id]
	if !ok {
		return 0, os.ErrNotExist
	}

	return time.Until(mission.Deadline), nil
}

type MissionSuggestion struct {
	Title    string    `json:"title"`
	Goal     string    `json:"goal"`
	Deadline time.Time `json:"deadline"`
	Reason   string    `json:"reason"`
}

func (m *Manager) ParseMission(text string, querier interface{ QueryWithContext(context.Context, string, string) (string, error) }) (*MissionSuggestion, error) {
	// ... (rest of ParseMission)
	return &suggestion, nil
}

func (m *Mission) BootstrapRequirements(querier interface{ QueryWithContext(context.Context, string, string) (string, error) }) error {
	prompt := fmt.Sprintf("MISSION: %s\nGOAL: %s\n\nBased on this mission, autonomously generate a shell script to bootstrap the project environment (e.g., create directories, initialize git/go/rust, create README.md). \n\nReturn SH SCRIPT ONLY. NO MARKDOWN.", m.Title, m.Goal)
	
	script, err := querier.QueryWithContext(context.Background(), prompt, "crud")
	if err != nil {
		return err
	}

	// Execution logic for the bootstrap script
	// In a real scenario, we'd want to be careful here, 
	// but for Auracrab, we execute with 'Strategic Confidence'.
	tmpFile := filepath.Join(os.TempDir(), "auracrab_bootstrap.sh")
	os.WriteFile(tmpFile, []byte(script), 0755)
	defer os.Remove(tmpFile)

	cmd := exec.Command("bash", tmpFile)
	return cmd.Run()
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &m.missions)
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.missions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}
