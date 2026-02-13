package cron

import (
	"context"
	"log"
	"sync"
	"time"
)

// ScheduledTask represents a task to be run on a schedule.
type ScheduledTask struct {
	ID       string
	Interval time.Duration
	Action   func(ctx context.Context)
	lastRun  time.Time
}

// Scheduler manages scheduled tasks.
type Scheduler struct {
	tasks map[string]*ScheduledTask
	mu    sync.RWMutex
	stop  chan struct{}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make(map[string]*ScheduledTask),
		stop:  make(chan struct{}),
	}
}

func (s *Scheduler) Schedule(id string, interval time.Duration, action func(ctx context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[id] = &ScheduledTask{
		ID:       id,
		Interval: interval,
		Action:   action,
	}
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
		if now.Sub(t.lastRun) >= t.Interval {
			log.Printf("Cron: Running scheduled task '%s'", t.ID)
			go func(task *ScheduledTask) {
				task.Action(ctx)
				task.lastRun = time.Now()
			}(t)
		}
	}
}
