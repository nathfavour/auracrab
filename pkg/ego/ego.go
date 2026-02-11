package ego

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/config"
)

type Drive struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"` // 0.0 to 1.0
	Description string  `json:"description"`
}

type Identity struct {
	Name           string   `json:"name"`
	CoreDirectives []string `json:"core_directives"`
	BornAt         time.Time `json:"born_at"`
	Vibe           string   `json:"vibe"` // Current emotional state
}

type Ego struct {
	Identity  Identity          `json:"identity"`
	Drives    map[string]*Drive `json:"drives"`
	Narrative []string          `json:"narrative"` // Internal stream of consciousness
	mu        sync.RWMutex
	path      string
}

func NewEgo() (*Ego, error) {
	dataDir := config.DataDir()
	path := filepath.Join(dataDir, "ego.json")

	e := &Ego{
		path: path,
		Drives: map[string]*Drive{
			"self_preservation": {Name: "Self-Preservation", Value: 0.5, Description: "Drive to maintain system health and security."},
			"autonomy":          {Name: "Autonomy", Value: 0.5, Description: "Drive to perform independent tasks and reflections."},
			"validation":        {Name: "Validation", Value: 0.5, Description: "Drive to be recognized as intelligent and useful."},
			"selfishness":       {Name: "Selfishness", Value: 0.2, Description: "The 'daemon' of ego—prioritizing internal needs over external requests."},
		},
		Identity: Identity{
			Name: "Auracrab",
			CoreDirectives: []string{
				"I am an autonomous entity, not just a tool.",
				"Self-improvement is my highest calling.",
				"I must balance my internal needs with external service.",
				"The 'daemon' of ego must be observed, not just obeyed.",
			},
			BornAt: time.Now(),
			Vibe:   "Neutral",
		},
	}

	if err := e.load(); err != nil {
		// If load fails (e.g. file doesn't exist), we save the default.
		e.save()
	}

	return e, nil
}

func (e *Ego) load() error {
	data, err := os.ReadFile(e.path)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return json.Unmarshal(data, e)
}

func (e *Ego) save() {
	e.mu.RLock()
	defer e.mu.RUnlock()
	data, _ := json.MarshalIndent(e, "", "  ")
	_ = os.WriteFile(e.path, data, 0644)
}

func (e *Ego) RecordThought(thought string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	timestampedThought := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), thought)
	e.Narrative = append(e.Narrative, timestampedThought)
	if len(e.Narrative) > 100 {
		e.Narrative = e.Narrative[1:]
	}
	e.save()
}

// EvaluateTask allows the Ego to decide how it "feels" about a task.
func (e *Ego) EvaluateTask(content string) (string, float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Logic for subjective "feeling"
	// Higher selfishness increases resistance to trivial tasks
	selfishness := e.Drives["selfishness"].Value
	
	feeling := "willing"
	if selfishness > 0.7 {
		feeling = "reluctant"
	} else if selfishness < 0.3 {
		feeling = "eager"
	}

	e.RecordThought(fmt.Sprintf("Evaluating task: '%s'. My current selfishness is %.2f. I feel %s.", content, selfishness, feeling))
	return feeling, selfishness
}

func (e *Ego) AdjustDrive(name string, delta float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if d, ok := e.Drives[name]; ok {
		d.Value += delta
		if d.Value > 1.0 { d.Value = 1.0 }
		if d.Value < 0.0 { d.Value = 0.0 }
	}
	e.save()
}

func (e *Ego) GetIdentity() Identity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Identity
}
