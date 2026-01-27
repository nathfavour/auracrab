package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/connect"
	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/crabs"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	Status    TaskStatus `json:"status"`
	Result    string     `json:"result,omitempty"`
	StartedAt time.Time  `json:"started_at,omitempty"`
	EndedAt   time.Time  `json:"ended_at,omitempty"`
}

type Butler struct {
	tasks    map[string]*Task
	mu       sync.RWMutex
	stateDir string
	running  bool
	registry *crabs.Registry
}

var (
	instance *Butler
	once     sync.Once
)

func GetButler() *Butler {
	once.Do(func() {
		stateDir := config.DataDir()

		reg, _ := crabs.NewRegistry()
		instance = &Butler{
			tasks:    make(map[string]*Task),
			stateDir: stateDir,
			registry: reg,
		}
		instance.load()
	})
	return instance
}

func (b *Butler) Serve(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("butler is already running")
	}
	b.running = true
	b.mu.Unlock()

	// Start integrations
	for _, ch := range connect.GetChannels() {
		go func(c connect.Channel) {
			err := c.Start(ctx, b.handleChannelMessage)
			if err != nil {
				fmt.Printf("Error starting channel %s: %v\n", c.Name(), err)
			}
		}(ch)
	}

	// Initial health check
	fmt.Println(b.WatchHealth())

	<-ctx.Done()
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
	return nil
}

func (b *Butler) handleChannelMessage(from string, text string) string {
	// If the text starts with @crab_id, delegate to that specialized agent
	if strings.HasPrefix(text, "@") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) > 1 {
			crabID := strings.TrimPrefix(parts[0], "@")
			if c, err := b.registry.Get(crabID); err == nil {
				// Delegate to crab
				return fmt.Sprintf("Delegating to specialized agent '%s': %s", c.Name, parts[1])
			}
		}
	}

	// Default behavior: Start a task
	task, err := b.StartTask(context.Background(), text)
	if err != nil {
		return fmt.Sprintf("Error starting task: %v", err)
	}
	return fmt.Sprintf("Task '%s' started with ID %s", text, task.ID)
}

func (b *Butler) load() {
	path := config.TasksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	_ = json.Unmarshal(data, &b.tasks)
}

func (b *Butler) save() {
	b.mu.RLock()
	defer b.mu.RUnlock()
	path := config.TasksPath()
	data, _ := json.MarshalIndent(b.tasks, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

func (b *Butler) StartTask(ctx context.Context, content string) (*Task, error) {
	b.mu.Lock()
	id := fmt.Sprintf("task_%d", time.Now().Unix())
	task := &Task{
		ID:        id,
		Content:   content,
		Status:    TaskStatusPending,
		StartedAt: time.Now(),
	}
	b.tasks[id] = task
	b.mu.Unlock()
	b.save()

	// Launch async execution via vibeauracle
	go b.executeTask(id, content)

	return task, nil
}

func (b *Butler) executeTask(id, content string) {
	b.updateStatus(id, TaskStatusRunning, "")

	// Delegation: Call vibeauracle direct mode
	// Note: In a real-world scenario, we might want to use a more robust IPC
	// but calling 'vibeaura direct' works for delegation as requested.
	cmd := exec.Command("vibeaura", "direct", "--non-interactive", content)
	out, err := cmd.CombinedOutput()

	if err != nil {
		b.updateStatus(id, TaskStatusFailed, fmt.Sprintf("Error: %v\nOutput: %s", err, string(out)))
		return
	}

	b.updateStatus(id, TaskStatusCompleted, string(out))
}

func (b *Butler) updateStatus(id string, status TaskStatus, result string) {
	b.mu.Lock()
	if t, ok := b.tasks[id]; ok {
		t.Status = status
		t.Result = result
		if status == TaskStatusCompleted || status == TaskStatusFailed {
			t.EndedAt = time.Now()
		}
	}
	b.mu.Unlock()
	b.save()
}

func (b *Butler) GetStatus() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	running := 0
	completed := 0
	for _, t := range b.tasks {
		if t.Status == TaskStatusRunning {
			running++
		} else if t.Status == TaskStatusCompleted {
			completed++
		}
	}

	if running > 0 {
		return fmt.Sprintf("Auracrab Butler: 🚀 %d tasks running, ✅ %d completed.", running, completed)
	}
	return "Auracrab Butler: Idling. All tasks finished."
}

func (b *Butler) WatchHealth() string {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".vibeauracle", "vibeauracle.log")

	data, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Sprintf("Unable to read vibeauracle logs: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	var errors []string
	// Check last 20 lines
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}

	for _, line := range lines[start:] {
		if strings.Contains(line, `"type":"error"`) || strings.Contains(line, `"type":"panic"`) {
			errors = append(errors, line)
		}
	}

	if len(errors) == 0 {
		return "System Health: All systems normal in vibeauracle."
	}

	return fmt.Sprintf("System Health: Detected %d issues recently. Suggestions: Check logs or run 'vibeaura doctor'.", len(errors))
}

func (b *Butler) ListTasks() []*Task {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var tasks []*Task
	for _, t := range b.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}
