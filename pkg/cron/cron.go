package cron
package cron

import (
	"context"
	"log"
	"sync"






































































}	}		}			}(t)				task.lastRun = time.Now()				task.Action(ctx)			go func(task *ScheduledTask) {			log.Printf("Cron: Running scheduled task '%s'", t.ID)		if now.Sub(t.lastRun) >= t.Interval {	for _, t := range s.tasks {	now := time.Now()	defer s.mu.RUnlock()	s.mu.RLock()func (s *Scheduler) checkTasks(ctx context.Context) {}	close(s.stop)func (s *Scheduler) Stop() {}	}		}			s.checkTasks(ctx)		case <-ticker.C:			return		case <-s.stop:			return		case <-ctx.Done():		select {	for {	defer ticker.Stop()	ticker := time.NewTicker(1 * time.Minute)func (s *Scheduler) Start(ctx context.Context) {}	}		Action:   action,		Interval: interval,		ID:       id,	s.tasks[id] = &ScheduledTask{	defer s.mu.Unlock()	s.mu.Lock()func (s *Scheduler) Schedule(id string, interval time.Duration, action func(ctx context.Context)) {}	}		stop:  make(chan struct{}),		tasks: make(map[string]*ScheduledTask),	return &Scheduler{func NewScheduler() *Scheduler {}	stop  chan struct{}	mu    sync.RWMutex	tasks map[string]*ScheduledTasktype Scheduler struct {// Scheduler manages scheduled tasks.}	lastRun  time.Time	Action   func(ctx context.Context)	Interval time.Duration	ID       stringtype ScheduledTask struct {// ScheduledTask represents a task to be run on a schedule.)	"time"