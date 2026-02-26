package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
)

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
)

type PulseStep struct {
	ID          string
	Description string
	Status      StepStatus
	Action      func(ctx context.Context) (string, error)
	Result      string
}

type PulseTask struct {
	ID        string
	Goal      string
	Steps     []*PulseStep
	Current   int
	Platform  string
	ChatID    string
	CreatedAt time.Time
}

type NervousSystem struct {
	mu      sync.RWMutex
	tasks   map[string]*PulseTask
	butler  *Butler
}

func NewNervousSystem(b *Butler) *NervousSystem {
	return &NervousSystem{
		tasks:  make(map[string]*PulseTask),
		butler: b,
	}
}

func (ns *NervousSystem) Name() string {
	return "NervousSystem"
}

// Plan creates a new recursive task based on a goal.
func (ns *NervousSystem) Plan(platform, chatID, goal string) *PulseTask {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	taskID := fmt.Sprintf("pulse_%d", time.Now().UnixNano())
	task := &PulseTask{
		ID:        taskID,
		Goal:      goal,
		Platform:  platform,
		ChatID:    chatID,
		CreatedAt: time.Now(),
		Steps:     []*PulseStep{},
	}
	ns.tasks[taskID] = task
	return task
}

// Pulse implements spine.Cell
func (ns *NervousSystem) Pulse(ctx context.Context) error {
	ns.mu.RLock()
	activeTasks := []*PulseTask{}
	for _, t := range ns.tasks {
		if t.Current < len(t.Steps) || len(t.Steps) == 0 {
			activeTasks = append(activeTasks, t)
		}
	}
	ns.mu.RUnlock()

	for _, task := range activeTasks {
		// 1. If task has no steps, it needs "Initial Planning"
		if len(task.Steps) == 0 {
			go ns.initialPlanning(ctx, task)
			continue
		}

		// 2. Execute the current step if it's pending
		step := task.Steps[task.Current]
		if step.Status == StepPending {
			go ns.executeStep(ctx, task, step)
		}
	}

	return nil
}

func (ns *NervousSystem) initialPlanning(ctx context.Context, task *PulseTask) {
	prompt := fmt.Sprintf("TASK_PLANNING: Goal: '%s'. Break this into 2-5 atomic, executable steps. Return a simple bulleted list of descriptions.", task.Goal)
	
	// Fast thinking
	res, err := ns.butler.QueryWithContext(ctx, prompt, "plan")
	if err != nil {
		return
	}

	// Simple parser for bullet points
	lines := []string{}
	for _, line := range append([]string{}, res) {
		if line != "" {
			lines = append(lines, line)
		}
	}

	ns.mu.Lock()
	for i, desc := range lines {
		task.Steps = append(task.Steps, &PulseStep{
			ID:          fmt.Sprintf("%s_s%d", task.ID, i),
			Description: desc,
			Status:      StepPending,
		})
	}
	ns.mu.Unlock()

	ns.butler.sendUpdate(task.Platform, task.ChatID, fmt.Sprintf("🧬 Pulse Plan for '%s' initialized with %d stages.", task.Goal, len(task.Steps)))
}

func (ns *NervousSystem) executeStep(ctx context.Context, task *PulseTask, step *PulseStep) {
	ns.mu.Lock()
	step.Status = StepRunning
	ns.mu.Unlock()

	biology.GetMetabolism().Burn(biology.CostComputeLow)

	// In a real implementation, the 'Action' would be a dynamic call.
	// For now, we use the Brain to execute the specific step description.
	prompt := fmt.Sprintf("TASK_EXECUTION: Goal: '%s'. Current Step: '%s'. Perform this step and return the result.", task.Goal, step.Description)
	
	res, err := ns.butler.QueryWithContext(ctx, prompt, "agent")
	
	ns.mu.Lock()
	if err != nil {
		step.Status = StepFailed
		step.Result = err.Error()
	} else {
		step.Status = StepCompleted
		step.Result = res
		task.Current++
	}
	isDone := task.Current >= len(task.Steps)
	ns.mu.Unlock()

	// "Lazy I/O" Progress update
	update := fmt.Sprintf("⚡ Step Complete: %s", step.Description)
	if isDone {
		update = fmt.Sprintf("✅ Goal Reached: %s\n\nFinal Outcome: %s", task.Goal, res)
	}
	ns.butler.sendUpdate(task.Platform, task.ChatID, update)
}
