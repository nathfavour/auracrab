package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
	"github.com/nathfavour/auracrab/pkg/memory"
	"github.com/nathfavour/auracrab/pkg/schema"
)

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
)

type NervousSystem struct {
	mu     sync.RWMutex
	butler *Butler
}

func NewNervousSystem(b *Butler) *NervousSystem {
	return &NervousSystem{
		butler: b,
	}
}

func (ns *NervousSystem) Name() string {
	return "NervousSystem"
}

// Pulse implements spine.Cell
func (ns *NervousSystem) Pulse(ctx context.Context) error {
	tasks := ns.butler.ListTasks()

	for _, task := range tasks {
		if task.Status != TaskStatusRunning && task.Status != TaskStatusPending {
			continue
		}

		if task.Continuity == nil {
			task.Continuity = &schema.TaskContinuity{
				Version: "1.0",
				TaskID:  task.ID,
				Goal:    task.Content,
				Status:  string(task.Status),
				Plan:    []schema.ContinuityStep{},
			}
		}

		// 1. If task has no steps, it needs "Initial Planning"
		if len(task.Continuity.Plan) == 0 {
			go ns.initialPlanning(ctx, task)
			continue
		}

		// 2. Execute the current step if it's pending
		if task.Continuity.Cursor < len(task.Continuity.Plan) {
			step := &task.Continuity.Plan[task.Continuity.Cursor]
			if step.Status == string(StepPending) {
				go ns.executeStep(ctx, task, step)
			}
		}
	}

	return nil
}

func (ns *NervousSystem) initialPlanning(ctx context.Context, task *Task) {
	// 1. Semantic Habituation: Check for cached plan
	if cachedSteps, ok := memory.GetHabitStore().Recall(task.Content); ok {
		fmt.Printf("NERVOUS: Habitual memory hit for '%s'! Reusing cached plan.\n", task.Content)
		ns.butler.mu.Lock()
		for i, desc := range cachedSteps {
			task.Continuity.Plan = append(task.Continuity.Plan, schema.ContinuityStep{
				ID:          fmt.Sprintf("%s_s%d", task.ID, i),
				Description: desc,
				Status:      string(StepPending),
			})
		}
		task.Continuity.RemainingSteps = cachedSteps
		ns.butler.mu.Unlock()
		ns.butler.save()
		ns.butler.SendUpdate(task.Platform, task.ChatID, fmt.Sprintf("🧠 Habitual memory triggered for '%s'. Pulse Plan recalled from experience.", task.Content))
		return
	}

	prompt := fmt.Sprintf("TASK_PLANNING: Goal: '%s'. Break this into 2-5 atomic, executable steps. Return a simple bulleted list of descriptions.", task.Content)

	// Update ThoughtSignature for planning
	ts := &ThoughtSignature{Goal: task.Content, PulseCount: task.Continuity.PulseCount}

	// Use metabolic query for planning
	resp, err := ns.butler.QueryMetabolic(ctx, prompt, "plan", ts, &Fovea{ActiveSkills: []string{"system"}})
	if err != nil {
		return
	}

	// Improved parser for bullet points
	lines := []string{}
	for _, line := range strings.Split(resp.Content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove common bullet prefixes
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		lines = append(lines, line)
	}

	ns.butler.mu.Lock()
	for i, desc := range lines {
		task.Continuity.Plan = append(task.Continuity.Plan, schema.ContinuityStep{
			ID:          fmt.Sprintf("%s_s%d", task.ID, i),
			Description: desc,
			Status:      string(StepPending),
		})
	}
	task.Continuity.RemainingSteps = lines
	ns.butler.mu.Unlock()
	ns.butler.save()

	ns.butler.SendUpdate(task.Platform, task.ChatID, fmt.Sprintf("🧬 Pulse Plan for '%s' initialized with %d stages.", task.Content, len(task.Continuity.Plan)))
}

func (ns *NervousSystem) executeStep(ctx context.Context, task *Task, step *schema.ContinuityStep) {
	ns.butler.mu.Lock()
	step.Status = string(StepRunning)
	task.Status = TaskStatusRunning
	ns.butler.mu.Unlock()

	biology.GetMetabolism().Burn(biology.CostComputeLow)

	// Foveated Sensing: Only activate skills relevant to the task if possible.
	fovea := &Fovea{
		ActiveSkills: []string{"system", "browser"},
	}

	ts := &ThoughtSignature{
		Goal:           task.Content,
		PulseCount:     task.Continuity.PulseCount,
		RemainingSteps: task.Continuity.RemainingSteps,
		Anomalies:      task.Continuity.Anomalies,
	}

	prompt := fmt.Sprintf("TASK_EXECUTION: Goal: '%s'. Current Step: '%s'. Perform this step and return the result.", task.Content, step.Description)

	resp, err := ns.butler.QueryMetabolic(ctx, prompt, "agent", ts, fovea)

	ns.butler.mu.Lock()
	task.Continuity.PulseCount++
	task.Continuity.LastCheckpoint = time.Now().Unix()
	if err != nil {
		step.Status = string(StepFailed)
		step.Result = err.Error()
		task.Continuity.Anomalies = append(task.Continuity.Anomalies, err.Error())
	} else {
		step.Status = string(StepCompleted)
		step.Result = resp.Content
		task.Continuity.Cursor++
		if len(task.Continuity.RemainingSteps) > 0 {
			task.Continuity.RemainingSteps = task.Continuity.RemainingSteps[1:]
		}
	}
	isDone := task.Continuity.Cursor >= len(task.Continuity.Plan)
	if isDone {
		task.Status = TaskStatusCompleted
		task.EndedAt = time.Now()
	}
	ns.butler.mu.Unlock()
	ns.butler.save()

	// "Lazy I/O" Progress update
	if isDone {
		// Semantic Habituation: Record successful plan
		allSteps := []string{}
		for _, s := range task.Continuity.Plan {
			allSteps = append(allSteps, s.Description)
		}
		memory.GetHabitStore().Learn(task.Content, allSteps)

		update := fmt.Sprintf("✅ Goal Reached: %s\n\nFinal Outcome: %s", task.Content, resp.Content)
		ns.butler.SendUpdateExt(task.Platform, task.ChatID, update, false) // Fast I/O for completion
	} else {
		update := fmt.Sprintf("⚡ Step Complete: %s", step.Description)
		ns.butler.SendUpdateExt(task.Platform, task.ChatID, update, true) // Lazy I/O for progress
	}
}
