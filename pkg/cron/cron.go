package cron

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// ScheduledTask represents a task to be run on a schedule.
type ScheduledTask struct {
	ID       string        `json:"id"`
	Interval time.Duration `json:"interval"`
	LastRun  time.Time     `json:"last_run"`
	Persistent bool        `json:"persistent"`
	Action   func(ctx context.Context) `json:"-"`
}

// Scheduler manages scheduled tasks.
type Scheduler struct {
	tasks map[string]*ScheduledTask
	mu    sync.RWMutex
	stop  chan struct{}
	path  string
}

func NewScheduler(path string) *Scheduler {
	s := &Scheduler{
		tasks: make(map[string]*ScheduledTask),
		stop:  make(chan struct{}),
		path:  path,
	}
	s.load()
	return s
}

func (s *Scheduler) Schedule(id string, interval time.Duration, action func(ctx context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if existing, ok := s.tasks[id]; ok {
		existing.Interval = interval
		existing.Action = action
		return
	}

	s.tasks[id] = &ScheduledTask{
		ID:       id,
		Interval: interval,
		Action:   action,
		Persistent: true,
	}
	s.save()
}

func (s *Scheduler) load() {
	if s.path == "" { return }
	data, err := os.ReadFile(s.path)
	if err != nil { return }
	
	var tasks map[string]*ScheduledTask
	if err := json.Unmarshal(data, &tasks); err == nil {
		s.tasks = tasks
	}
}

func (s *Scheduler) save() {
	if s.path == "" { return }
	data, _ := json.MarshalIndent(s.tasks, "", "  ")
	_ = os.WriteFile(s.path, data, 0644)
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stop:
			return
		case <-ticker.C:
			s.checkTasks(ctx)
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) checkTasks(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, t := range s.tasks {
		if t.Action == nil {
			continue
		}
		if now.Sub(t.LastRun) >= t.Interval {
			log.Printf("Cron: Running scheduled task '%s'", t.ID)
			go func(task *ScheduledTask) {
				task.Action(ctx)
				s.mu.Lock()
				task.LastRun = time.Now()
				s.mu.Unlock()
				s.save()
			}(t)
		}
	}
}
