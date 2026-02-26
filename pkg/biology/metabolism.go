package biology

import (
	"sync"
	"time"
)

// MetabolicRate defines the energy cost of various actions.
type MetabolicRate float64

const (
	CostAPIQuery   MetabolicRate = 0.05 // High energy: requires network and LLM
	CostDiskWrite  MetabolicRate = 0.01 // Low energy: local IO
	CostComputeLow MetabolicRate = 0.005 // Low energy: simple logic
	CostComputeHigh MetabolicRate = 0.08 // Extreme energy: compilation or heavy analysis
)

// Metabolism tracks the energy consumption over time.
type Metabolism struct {
	mu           sync.RWMutex
	TotalBurned  float64
	StartTime    time.Time
	LastActivity time.Time
}

var (
	metabolismInstance *Metabolism
	metOnce            sync.Once
)

func GetMetabolism() *Metabolism {
	metOnce.Do(func() {
		metabolismInstance = &Metabolism{
			StartTime: time.Now(),
		}
	})
	return metabolismInstance
}

// Burn consumes energy for an action.
func (m *Metabolism) Burn(rate MetabolicRate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalBurned += float64(rate)
	m.LastActivity = time.Now()
}

// GetStats returns the metabolic summary.
func (m *Metabolism) GetStats() (total float64, uptime time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.TotalBurned, time.Since(m.StartTime)
}

// Efficiency calculates work done vs energy burned (simplified).
func (m *Metabolism) Efficiency(tasksCompleted int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.TotalBurned == 0 {
		return 1.0
	}
	return float64(tasksCompleted) / m.TotalBurned
}
