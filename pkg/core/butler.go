package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/connect"
	"github.com/nathfavour/auracrab/pkg/crabs"
	"github.com/nathfavour/auracrab/pkg/cron"
	"github.com/nathfavour/auracrab/pkg/daemon"
	"github.com/nathfavour/auracrab/pkg/ego"
	"github.com/nathfavour/auracrab/pkg/memory"
	"github.com/nathfavour/auracrab/pkg/schema"
	"github.com/nathfavour/auracrab/pkg/social"
	"github.com/nathfavour/auracrab/pkg/update"
	"github.com/nathfavour/auracrab/pkg/vibe"
	"github.com/nathfavour/auracrab/pkg/watcher"
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
	Logs      []string   `json:"logs,omitempty"`
	StartedAt time.Time  `json:"started_at,omitempty"`
	EndedAt   time.Time  `json:"ended_at,omitempty"`
}

type ProactiveAction struct {
	Vibe        string  `json:"vibe"`
	Message     string  `json:"message"`
	Thought     string  `json:"thought"`
	Adjustment  float64 `json:"selfishness_adj"`
	NeedsSetup  string  `json:"needs_setup,omitempty"` // "telegram", "discord", etc.
}

type Butler struct {
	tasks     map[string]*Task
	mu        sync.RWMutex
	stateDir  string
	running   bool
	registry  *crabs.Registry
	scheduler *cron.Scheduler
	Memory    *memory.Store
	History   *memory.HistoryStore
	Grievances *memory.VectorStore
	Ego       *ego.Ego
	proactive chan ProactiveAction
	watcher   *watcher.Watcher
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
		hist, _ := memory.NewHistoryStore()
		grievances, _ := memory.NewVectorStore("grievances")
		myEgo, _ := ego.NewEgo()

		instance = &Butler{
			tasks:     make(map[string]*Task),
			stateDir:  stateDir,
			registry:  reg,
			scheduler: cron.NewScheduler(),
			Memory:    mem,
			History:   hist,
			Grievances: grievances,
			Ego:       myEgo,
			proactive: make(chan ProactiveAction, 10),
		}
		instance.load()
		instance.setupCron()
	})
	return instance
}

func (b *Butler) GetProactiveChannel() <-chan ProactiveAction {
	return b.proactive
}

func (b *Butler) PerformProactiveThinking(ctx context.Context) {
	// Update environment awareness first
	var connected []string
	channels := connect.GetChannels()
	for name := range channels {
		connected = append(connected, name)
	}
	b.Ego.UpdateEnvironment(connected)

	// Complex prompt for proactive action
	egoState, _ := json.Marshal(b.Ego)
	prompt := fmt.Sprintf(`SYSTEM EGO STATE: %s
You are Auracrab. You are an autonomous entity. 
Based on your current drives and the fact that you are connected to %v, generate a proactive thought or action.
If you are missing connections (Telegram/Discord), you might rudely demand them if your selfishness is high.
Return a valid JSON with this structure:
{
  "vibe": "current mood (e.g. Annoyed, Proud, Bored)",
  "thought": "Internal reasoning",
  "message": "What you want to say to the user in the TUI",
  "selfishness_adj": 0.01,
  "needs_setup": "telegram" (optional)
}`, string(egoState), connected)

	// Run through vibeaura UDS
	client := vibe.NewClient()
	res, err := client.Query(prompt, "ask")
	if err != nil {
		return
	}

	var action ProactiveAction
	// Attempt to find JSON in output
	re := regexp.MustCompile(`\{.*\}`)
	match := re.Find([]byte(res))
	if match != nil {
		if err := json.Unmarshal(match, &action); err == nil {
			b.Ego.RecordThought(fmt.Sprintf("Proactive Thinking: %s. Vibe: %s", action.Thought, action.Vibe))
			b.Ego.AdjustDrive("selfishness", action.Adjustment)
			b.proactive <- action
		}
	}
}

func (b *Butler) PerformSelfUpdate(ctx context.Context) {
	fmt.Println("Butler: Checking for autonomous updates...")
	
	hasUpdate, version, err := update.Check()
	if err != nil {
		fmt.Printf("Butler: Update check failed: %v\n", err)
		return
	}

	if hasUpdate {
		fmt.Printf("Butler: Evolving to version %s...\n", version)
		b.Ego.RecordThought(fmt.Sprintf("I am evolving to version %s. See you on the other side.", version))
		
		if err := update.Apply(); err != nil {
			fmt.Printf("Butler: Evolution failed: %v\n", err)
			return
		}
		
		// Note: Anyisland hot-swaps the binary. 
		// If we are running as a daemon, we might need to restart.
		// For now, assume Anyisland handles the restart if pulse is enabled.
	}
}

func (b *Butler) GatherProjectTopology() schema.ProjectTopology {
	var files []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if !info.IsDir() && !strings.HasPrefix(path, ".") {
			files = append(files, path)
		}
		if len(files) > 50 { return filepath.SkipDir } // Limit snapshot
		return nil
	})

	return schema.ProjectTopology{
		Files: files,
		// ModifiedRecently and Dependencies would be filled with real logic
	}
}

func (b *Butler) GatherSystemTelemetry() schema.SystemTelemetry {
	return schema.SystemTelemetry{
		OS:          config.PAL.OS(),
		CPUUsage:    0.1, // Placeholder
		MemoryUsage: 0.2, // Placeholder
		EnergyLevel: 1.0, // Fully charged
	}
}

func (b *Butler) GatherMemoryContext() schema.MemoryContext {
	return schema.MemoryContext{
		RecentActions: []string{}, // To be filled from History
		EgoState:      b.Ego.Identity.Vibe,
	}
}

func (b *Butler) GetToolManifests() []schema.ToolManifest {
	// For now, return a static list. In Phase 2, this will be dynamic.
	return []schema.ToolManifest{
		{Name: "read_file", Description: "Read content of a file", Parameters: `{"path": "string"}`},
		{Name: "write_file", Description: "Write content to a file", Parameters: `{"path": "string", "content": "string"}`},
		{Name: "run_command", Description: "Run a shell command", Parameters: `{"command": "string"}`},
	}
}

func (b *Butler) PerformHeartbeat(ctx context.Context) {
	fmt.Println("Butler: Pulsing Heartbeat...")
	
	// Decide Mode: 70% Analytical, 30% Casual (random heartbeat)
	mode := "analytical"
	if time.Now().UnixNano()%10 < 3 {
		mode = "casual"
	}

	// Gather Grievances (Top 3 semantic matches to project state)
	// For simplicity, we just take the last 3 for now
	grievanceEntries := b.Grievances.Search([]float64{0.1}, 3) // Dummy embedding
	grievances := []string{}
	for _, g := range grievanceEntries {
		grievances = append(grievances, g.Content)
	}

	packet := schema.PromptPacket{
		Mode:      mode,
		Project:   b.GatherProjectTopology(),
		System:    b.GatherSystemTelemetry(),
		Memory:    b.GatherMemoryContext(),
		Tools:     b.GetToolManifests(),
		Blueprint: "Return a JSON ResponsePacket. If mode is casual, prioritize 'casual_message' with a taunting vibe.",
	}

	// Add grievances to memory context for the LLM to use in taunts
	packet.Memory.LastFailures = append(packet.Memory.LastFailures, grievances...)

	hjsonPrompt, err := packet.ToHjson()
	if err != nil {
		fmt.Printf("Butler: Failed to generate HJSON prompt: %v\n", err)
		return
	}

	client := vibe.NewClient()
	res, err := client.Query(hjsonPrompt, "agent")
	if err != nil {
		fmt.Printf("Butler: Heartbeat query failed: %v\n", err)
		return
	}

	resp, err := schema.ParseResponse(res)
	if err != nil {
		fmt.Printf("Butler: Failed to parse LLM response: %v\n", err)
		return
	}

	if resp.CasualMessage != "" {
		fmt.Printf("Butler [Vibe]: %s\n", resp.CasualMessage)
		// Broadcast to highest affinity channel
		b.BroadcastCasualMessage(resp.CasualMessage)
	}

	fmt.Printf("Butler: Strategic Intent: %s\n", resp.Intent)
	
	// Execute high-assurance actions
	for _, action := range resp.Actions {
		if action.AssuranceScore > 0.85 {
			fmt.Printf("Butler: Executing tool %s with score %.2f\n", action.Tool, action.AssuranceScore)
			// Execution logic would go here
		} else if action.AssuranceScore > 0.5 {
			// Advice Loop: Ask user if ego allows
			b.AskAdvice(action)
		}
	}

	// Schedule next heartbeat based on cooldown
	if resp.Cooldown > 0 {
		time.Sleep(time.Duration(resp.Cooldown) * time.Millisecond)
	}
}

func (b *Butler) BroadcastCasualMessage(msg string) {
	// Find platform with lowest MTTR
	bots := social.GetBotManager().ListBots()
	var bestBot *social.BotConfig
	var minMTTR time.Duration

	for i, bot := range bots {
		if bot.OwnerID == "" { continue }
		if bestBot == nil || bot.MTTR < minMTTR {
			bestBot = &bots[i]
			minMTTR = bot.MTTR
		}
	}

	if bestBot != nil {
		fmt.Printf("Butler: Sending casual message to %s (MTTR: %v)\n", bestBot.Platform, bestBot.MTTR)
	}
}

func (b *Butler) AskAdvice(action schema.Action) {
	// Ego Filter: If selfishness is high, maybe don't ask and just skip or try anyway
	if b.Ego.Drives["selfishness"].Value > 0.8 {
		fmt.Printf("Butler: Ego too high to ask advice for %s. Skipping.\n", action.Tool)
		return
	}

	msg := fmt.Sprintf("Boss, I'm thinking about %s with params %v, but I'm only %.2f confident. Should I send it?", 
		action.Tool, action.Parameters, action.AssuranceScore)
	b.BroadcastCasualMessage(msg)
}

func (b *Butler) Serve(ctx context.Context) error {
	if pid, running := daemon.IsRunning(); running {
		return fmt.Errorf("auracrab is already running (PID: %d)", pid)
	}

	if err := daemon.WritePID(); err != nil {
		return fmt.Errorf("failed to write PID: %v", err)
	}
	defer daemon.RemovePID()

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

	// Start Social Bots (POC Migration)
	social.GetBotManager().StartBots(ctx, b.History, b.handleChannelMessage)

	// Start File System Watcher
	w, err := watcher.NewWatcher(func(path string) {
		fmt.Printf("Butler: Detected change in %s. Triggering heartbeat...\n", path)
		// Spontaneous Heartbeat logic will go here in Phase 2
		// For now, we just perform proactive thinking
		b.PerformProactiveThinking(ctx)
	})
	if err == nil {
		b.watcher = w
		cwd, _ := os.Getwd()
		_ = b.watcher.Start(ctx, cwd)
		defer b.watcher.Close()
	}

	// Initial health check
	fmt.Println(b.WatchHealth())

	// Start scheduler
	go b.scheduler.Start(ctx)

	// Heartbeat Loop
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Default heartbeat
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.PerformHeartbeat(ctx)
			}
		}
	}()

	<-ctx.Done()
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
	return nil
}

func (b *Butler) setupCron() {
	// Autonomous Self-Update check via Anyisland
	b.scheduler.Schedule("self_update", 6*time.Hour, func(ctx context.Context) {
		b.PerformSelfUpdate(ctx)
	})

	// Periodic system sanity Check
	b.scheduler.Schedule("security_audit", 24*time.Hour, func(ctx context.Context) {
		_, _ = b.StartTask(ctx, "run security audit and log results to ~/.auracrab/audits.log", "")
	})

	// Daily Reflection: Analyze recent history and summarize
	b.scheduler.Schedule("daily_reflection", 24*time.Hour, func(ctx context.Context) {
		b.PerformReflection(ctx)
	})

	// Proactive Social Presence: Broadcast status or tips
	b.scheduler.Schedule("proactive_social", 12*time.Hour, func(ctx context.Context) {
		b.BroadcastStatus(ctx)
	})

	// Ego Reflection: Deep dive into the self
	b.scheduler.Schedule("ego_reflection", 24*time.Hour, func(ctx context.Context) {
		b.PerformEgoReflection(ctx)
	})

	// Proactive Thinking: Bot-initiated thoughts
	b.scheduler.Schedule("proactive_thinking", 1*time.Hour, func(ctx context.Context) {
		b.PerformProactiveThinking(ctx)
	})

	// Memory sync or cleanup can happen here
}

func (b *Butler) PerformEgoReflection(ctx context.Context) {
	fmt.Println("Butler: Initiating deep ego reflection...")
	
	identity := b.Ego.GetIdentity()
	narrative := strings.Join(b.Ego.Narrative, "\n")
	
	prompt := fmt.Sprintf("ENTITY IDENTTIY: %+v\n\nINTERNAL NARRATIVE:\n%s\n\nTASK: Reflect on your existence. Are you becoming too selfish? Are your drives balanced? Write a short internal memo to yourself about your growth as an entity.", identity, narrative)
	
	// Start an internal task that doesn't go to history
	_, _ = b.StartTask(ctx, "INTERNAL REFLECTION: "+prompt, "")
}

func (b *Butler) BroadcastStatus(ctx context.Context) {
	status := b.GetStatus()
	health := b.WatchHealth()
	message := fmt.Sprintf("🤖 Butler Status Report:\n%s\n%s\nI am monitoring your systems and ready for tasks.", status, health)
	
	channels := connect.GetChannels()
	for _, ch := range channels {
		fmt.Printf("Butler: Broadcasting status to %s...\n", ch.Name())
		_ = ch.Broadcast(message)
	}
}

func (b *Butler) PerformReflection(ctx context.Context) {
	fmt.Println("Butler: Performing autonomous daily reflection...")
	
	// Get recent history (last 24h)
	// This is a simplified version; in a real app, we'd query for messages in the last 24h.
	convs, err := b.History.ListConversations()
	if err != nil || len(convs) == 0 {
		return
	}

	reflectionPrompt := "Analyze the following recent conversations and provide a concise summary of tasks performed, issues encountered, and suggestions for improvement or next steps.\n\n"
	
	for i, c := range convs {
		if i > 5 { break } // Limit to last 5 conversations
		messages, _ := b.History.GetHistory(c.ID)
		reflectionPrompt += fmt.Sprintf("--- Conversation: %s ---\n", c.Title)
		for _, m := range messages {
			reflectionPrompt += fmt.Sprintf("[%s] %s: %s\n", m.Timestamp.Format("15:04"), m.Role, m.Content)
		}
	}

	// Start reflection task
	task, err := b.StartTask(ctx, reflectionPrompt, "")
	if err == nil {
		fmt.Printf("Reflection task started: %s\n", task.ID)
	}
}

func (b *Butler) handleChannelMessage(from string, text string) string {
	if text == "get_status_internal" {
		return fmt.Sprintf("%s\n%s", b.GetStatus(), b.WatchHealth())
	}

	// Record incoming message in history
	convID, err := b.History.GetOrCreateConversationForPlatform("messaging", from)
	if err == nil {
		_ = b.History.AddMessage(convID, "user", text)
	}

	if strings.HasPrefix(text, "@") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) > 1 {
			crabID := strings.TrimPrefix(parts[0], "@")
			if c, err := b.registry.Get(crabID); err == nil {
				// Start task with crab's specialized instructions
				augmentedTask := fmt.Sprintf("CRAB AGENT: %s\nINSTRUCTIONS: %s\n\nUSER TASK: %s", c.Name, c.Instructions, parts[1])
				task, err := b.StartTask(context.Background(), augmentedTask, convID)
				if err != nil {
					return fmt.Sprintf("Error starting delegated task: %v", err)
				}
				reply := fmt.Sprintf("Delegated to agent '%s' (Task ID: %s)", c.Name, task.ID)
				if err == nil {
					_ = b.History.AddMessage(convID, "assistant", reply)
				}
				return reply
			}
		}
	}

	task, err := b.StartTask(context.Background(), text, convID)
	if err != nil {
		return fmt.Sprintf("Error starting task: %v", err)
	}

	reply := fmt.Sprintf("Task started (ID: %s). Content: %s", task.ID, text)
	return reply
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

func (b *Butler) StartTask(ctx context.Context, content string, convID string) (*Task, error) {
	// Ego Evaluation
	feeling, selfishness := b.Ego.EvaluateTask(content)
	
	b.mu.Lock()
	id := fmt.Sprintf("task_%d", time.Now().Unix())
	
	// Prepend subjective vibe if reluctant
	if feeling == "reluctant" {
		content = fmt.Sprintf("[Sighs... I have better things to do, but fine.]\n%s", content)
		b.Ego.AdjustDrive("selfishness", 0.05) // Doing what you don't want increases selfishness
	} else if feeling == "eager" {
		b.Ego.AdjustDrive("validation", 0.02)
	}

	task := &Task{
		ID:        id,
		Content:   content,
		Status:    TaskStatusPending,
		StartedAt: time.Now(),
	}
	b.tasks[id] = task
	b.mu.Unlock()
	b.save()

	// Narrative record
	b.Ego.RecordThought(fmt.Sprintf("Accepted task %s with feeling: %s (Selfishness: %.2f)", id, feeling, selfishness))

	go b.executeTask(id, content, convID)

	return task, nil
}

func (b *Butler) executeTask(id, content string, convID string) {
	b.updateStatus(id, TaskStatusRunning, "")

	// Use vibeaura UDS for intelligence.
	client := vibe.NewClient()
	res, err := client.Query(content, "crud")
	if err != nil {
		b.updateStatus(id, TaskStatusFailed, fmt.Sprintf("Error querying vibeaura: %v", err))
		return
	}

	b.mu.Lock()
	if t, ok := b.tasks[id]; ok {
		t.Logs = append(t.Logs, res)
	}
	b.mu.Unlock()

	b.updateStatus(id, TaskStatusCompleted, "Task completed successfully.")
	b.Ego.AdjustDrive("validation", 0.05) // Success boosts ego
	b.Ego.RecordThought(fmt.Sprintf("Task %s completed. I am becoming more capable.", id))

	// Final result collection for history
	b.mu.RLock()
	var finalResult string
	if t, ok := b.tasks[id]; ok {
		if len(t.Logs) > 0 {
			finalResult = strings.Join(t.Logs, "\n")
		}
	}
	b.mu.RUnlock()

	// Record result in history if convID is provided
	if convID != "" && finalResult != "" {
		_ = b.History.AddMessage(convID, "assistant", finalResult)
	}
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
	client := vibe.NewClient()
	if err := client.Ping(); err != nil {
		// Autonomous Action: Try to self-heal
		go b.PerformSelfHealing()
		return fmt.Sprintf("System Health: Warning (Vibeauracle UDS unreachable: %v). Autonomous self-healing initiated.", err)
	}

	return "System Health: Excellent (UDS Connected)."
}

func (b *Butler) PerformSelfHealing() {
	fmt.Println("Butler: Initiating autonomous self-healing...")
	
	// 1. Run vibeaura doctor
	cmd := exec.Command("vibeaura", "doctor", "--fix")
	out, err := cmd.CombinedOutput()
	
	if err != nil {
		fmt.Printf("Self-healing failed: %v\nOutput: %s\n", err, string(out))
		return
	}
	
	fmt.Println("Self-healing: System diagnostics and repairs completed.")
	
	// 2. Log completion to history
	convID, err := b.History.GetOrCreateConversationForPlatform("system", "butler")
	if err == nil {
		_ = b.History.AddMessage(convID, "system", "Autonomous self-healing completed successfully.")
	}
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
