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

"github.com/nathfavour/auracrab/pkg/config"
"github.com/nathfavour/auracrab/pkg/connect"
"github.com/nathfavour/auracrab/pkg/crabs"
"github.com/nathfavour/auracrab/pkg/cron"
"github.com/nathfavour/auracrab/pkg/memory"
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
	tasks     map[string]*Task
	mu        sync.RWMutex
	stateDir  string
	running   bool
	registry  *crabs.Registry
	scheduler *cron.Scheduler
	Memory    *memory.Store
}

var (
instance *Butler
once     sync.Once
)

func GetButler() *Butler {
	once.Do(func() {
		stateDir := config.DataDir()

		reg, _ := crabs.NewRegistry()
		mem, _ := memory.NewStore("global")
		instance = &Butler{
			tasks:     make(map[string]*Task),
			stateDir:  stateDir,
			registry:  reg,
			scheduler: cron.NewScheduler(),
			Memory:    mem,
		}
		instance.load()
		instance.setupCron()
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
	channels := connect.GetChannels()
	if len(channels) == 0 {
		fmt.Println("Butler: No messaging channels (Telegram/Discord) configured.")
	}

	for _, ch := range channels {
		go func(c connect.Channel) {
			err := c.Start(ctx, b.handleChannelMessage)
			if err != nil {
				fmt.Printf("Error starting channel %s: %v\n", c.Name(), err)
			}
		}(ch)
	}

	// Initial health check
	fmt.Println(b.WatchHealth())

	// Start scheduler
	go b.scheduler.Start(ctx)

	<-ctx.Done()
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
	return nil
}

func (b *Butler) setupCron() {
	// Periodic system sanity Check
	b.scheduler.Schedule("security_audit", 24*time.Hour, func(ctx context.Context) {
_, _ = b.StartTask(ctx, "run security audit and log results to ~/.auracrab/audits.log")
})

	// Memory sync or cleanup can happen here
}

func (b *Butler) handleChannelMessage(from string, text string) string {
	if text == "get_status_internal" {
		return fmt.Sprintf("%s\n%s", b.GetStatus(), b.WatchHealth())
	}

	if strings.HasPrefix(text, "@") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) > 1 {
			crabID := strings.TrimPrefix(parts[0], "@")
			if c, err := b.registry.Get(crabID); err == nil {
				return fmt.Sprintf("[%s] Delegating to agent '%s': %s", from, c.Name, parts[1])
			}
		}
	}

	task, err := b.StartTask(context.Background(), text)
	if err != nil {
		return fmt.Sprintf("Error starting task: %v", err)
	}
	return fmt.Sprintf("Task started (ID: %s). Content: %s", task.ID, text)
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

	go b.executeTask(id, content)

	return task, nil
}

func (b *Butler) executeTask(id, content string) {
	b.updateStatus(id, TaskStatusRunning, "")

	// Use vibeaura for intelligence
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
	return fmt.Sprintf("Auracrab: %d active, %d tasks done. System: Stable.", running, completed)
}

func (b *Butler) WatchHealth() string {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".vibeauracle", "vibeauracle.log")

	data, err := os.ReadFile(logPath)
	if err != nil {
		return "System Health: OK (Vibeauracle logs not found, assuming fresh start)"
	}

	lines := strings.Split(string(data), "\n")
	var errCount int
	start := len(lines) - 50
	if start < 0 {
		start = 0
	}

	for _, line := range lines[start:] {
		if strings.Contains(line, "error") || strings.Contains(line, "panic") {
			errCount++
		}
	}

	if errCount == 0 {
		return "System Health: Excellent."
	}
	return fmt.Sprintf("System Health: Warning (%d anomalies detected). Recommend 'vibeaura doctor'.", errCount)
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
