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
	"github.com/nathfavour/auracrab/pkg/ego"
	"github.com/nathfavour/auracrab/pkg/memory"
	"github.com/nathfavour/auracrab/pkg/mission"
	"github.com/nathfavour/auracrab/pkg/social"
	"github.com/nathfavour/auracrab/pkg/vibe"
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

type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

type QueuedMessage struct {
	Platform string
	ChatID   string
	From     string
	Text     string
	Priority Priority
	Received time.Time
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
	Missions  *mission.Manager
	Ego       *ego.Ego
	channels  map[string]connect.Channel
	
	highQueue   chan QueuedMessage
	normalQueue chan QueuedMessage
	lowQueue    chan QueuedMessage
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
		miss, _ := mission.NewManager()
		eg, _ := ego.NewEgo()

		instance = &Butler{
			tasks:       make(map[string]*Task),
			stateDir:    stateDir,
			registry:    reg,
			scheduler:   cron.NewScheduler(),
			Memory:      mem,
			History:     hist,
			Missions:    miss,
			Ego:         eg,
			channels:    make(map[string]connect.Channel),
			highQueue:   make(chan QueuedMessage, 50),
			normalQueue: make(chan QueuedMessage, 100),
			lowQueue:    make(chan QueuedMessage, 100),
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
	chans := connect.GetChannels()
	if len(chans) == 0 {
		fmt.Println("Butler: No messaging channels (Telegram/Discord) configured.")
	}

	b.mu.Lock()
	for name, ch := range chans {
		b.channels[name] = ch
	}
	b.mu.Unlock()

	for _, ch := range chans {
		go func(c connect.Channel) {
			err := c.Start(ctx, b.handleChannelMessage)
			if err != nil {
				fmt.Printf("Error starting channel %s: %v\n", c.Name(), err)
			}
		}(ch)
	}

	// Start Social Bots (POC Migration)
	social.GetBotManager().StartBots(ctx, b.History, b, b.handleChannelMessage)

	// Start queue processor
	go b.processQueue(ctx)

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
		_, _ = b.StartTask(context.Background(), "run security audit and log results to ~/.auracrab/audits.log", "")
	})

	// Memory sync or cleanup can happen here
}

func (b *Butler) QueryWithContext(ctx context.Context, prompt string, intent string) (string, error) {
	if intent == "" {
		intent = "vibe"
	}

	fullPrompt := b.buildPrompt(prompt, "")

	client := vibe.NewClient()
	reply, err := client.Query(fullPrompt, intent)
	if err != nil {
		// Fallback to HeuristicSynthesizer if vibeauracle is offline
		fmt.Printf("Vibeauracle error, using heuristic fallback: %v\n", err)
		heuristic := vibe.NewHeuristicSynthesizer()
		return heuristic.Synthesize(prompt), nil
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("empty response from vibeauracle")
	}
	return reply, nil
}

func (b *Butler) buildPrompt(userMessage string, historyText string) string {
	cwd, _ := os.Getwd()
	files, _ := filepath.Glob("*")
	if len(files) > 25 {
		files = files[:25]
	}
	dirSnapshot := strings.Join(files, "\n")
	if dirSnapshot == "" {
		dirSnapshot = "(no files discovered)"
	}

	prompt := fmt.Sprintf(
		"AURACRAB_SYSTEM_CONTEXT\nWORKING_DIRECTORY: %s\nPROJECT_FILES_SNAPSHOT:\n%s\n\nCONVERSATION_HISTORY:\n%s\n\nUSER_PROMPT:\n%s\n\nOUTPUT_RULES:\n- Return the final actionable answer only.\n- Do not include chain-of-thought or hidden reasoning.\n- Be concrete, execution-oriented, and directly useful.",
		cwd,
		dirSnapshot,
		historyText,
		userMessage,
	)
	fmt.Printf("Butler: Built prompt (length: %d)\n", len(prompt))
	return prompt
}

func (b *Butler) handleChannelMessage(platform, chatID, from, text string) string {
	fmt.Printf("Butler: Received message from %s on %s: %s\n", from, platform, text)
	if text == "get_status_internal" {
		return b.GetStatus() + "\n" + b.WatchHealth()
	}

	priority := PriorityNormal
	if strings.Contains(strings.ToLower(text), "urgent") || strings.Contains(strings.ToLower(text), "critical") {
		priority = PriorityHigh
	}

	msg := QueuedMessage{
		Platform: platform,
		ChatID:   chatID,
		From:     from,
		Text:     text,
		Priority: priority,
		Received: time.Now(),
	}

	switch priority {
	case PriorityHigh, PriorityCritical:
		b.highQueue <- msg
	case PriorityNormal:
		b.normalQueue <- msg
	default:
		b.lowQueue <- msg
	}

	return ""
}

func (b *Butler) processQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-b.highQueue:
			go b.processMessage(msg)
		case msg := <-b.normalQueue:
			go b.processMessage(msg)
		case msg := <-b.lowQueue:
			go b.processMessage(msg)
		case <-time.After(100 * time.Millisecond):
			// Keep loop alive
		}
	}
}

func (b *Butler) processMessage(msg QueuedMessage) {
	fmt.Printf("Butler: Processing message from %s [%s]: %s\n", msg.From, msg.Platform, msg.Text)

	// Send "Thinking..." heartbeat
	b.sendUpdate(msg.Platform, msg.ChatID, "Thinking...")

	// Record incoming message in history
	convID, err := b.History.GetOrCreateConversationForPlatform(msg.Platform, msg.ChatID)
	
	// Fetch conversation history for context BEFORE adding current message
	historyText := ""
	if convID != "" {
		history, _ := b.History.GetHistory(convID)
		// Only take last 10 messages for context window safety
		start := len(history) - 10
		if start < 0 {
			start = 0
		}
		for _, m := range history[start:] {
			historyText += fmt.Sprintf("%s: %s\n", strings.ToUpper(m.Role), m.Content)
		}
	}

	// Now add the current message to history
	if err == nil {
		_ = b.History.AddMessage(convID, "user", msg.Text)
	}

	var reply string
	if strings.HasPrefix(msg.Text, "@") {
		parts := strings.SplitN(msg.Text, " ", 2)
		if len(parts) > 1 {
			crabID := strings.TrimPrefix(parts[0], "@")
			if c, err := b.registry.Get(crabID); err == nil {
				// Start task with crab's specialized instructions
				augmentedTask := fmt.Sprintf("CRAB AGENT: %s\nINSTRUCTIONS: %s\n\nUSER TASK: %s", c.Name, c.Instructions, parts[1])
				task, err := b.StartTask(context.Background(), augmentedTask, convID)
				if err != nil {
					reply = fmt.Sprintf("Error starting delegated task: %v", err)
				} else {
					reply = fmt.Sprintf("Delegated to agent '%s' (Task ID: %s)", c.Name, task.ID)
				}
			}
		}
	}

	if reply == "" {
		// Context with hard timeout for the entire processing
		_, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		
		intent := "ask" // Default to documented 'ask' intent
		lowerText := strings.ToLower(msg.Text)
		if strings.Contains(lowerText, "plan") || strings.Contains(lowerText, "analyze") || strings.Contains(lowerText, "status") {
			intent = "plan"
		}
		if strings.HasPrefix(lowerText, "create") || strings.HasPrefix(lowerText, "update") || 
		   strings.HasPrefix(lowerText, "delete") || strings.HasPrefix(lowerText, "list") ||
		   strings.Contains(lowerText, "task") || strings.Contains(lowerText, "run") {
			intent = "crud"
		}

		promptWithContext := b.buildPrompt(msg.Text, historyText)

		client := vibe.NewClient()
		// Reverting to synchronous Query for better reliability as streaming isn't documented for UDS
		res, err := client.Query(promptWithContext, intent)
		if err != nil {
			reply = fmt.Sprintf("Error from vibeauracle: %v", err)
		} else {
			reply = res
		}

		if reply == "" {
			reply = "Empty response from vibeauracle."
		}
	}

	if convID != "" {
		_ = b.History.AddMessage(convID, "assistant", reply)
	}

	// Send final reply
	fmt.Printf("Butler: Sending reply to %s: %s\n", msg.From, reply)
	b.sendUpdate(msg.Platform, msg.ChatID, reply)
}

func (b *Butler) sendUpdate(platform, chatID, text string) {
	b.mu.RLock()
	ch, ok := b.channels[platform]
	b.mu.RUnlock()

	if ok {
		err := ch.Send(chatID, text)
		if err == nil {
			return
		}
		fmt.Printf("Butler: Channel Send failed for %s, falling back: %v\n", platform, err)
	}

	// Fallback to social bot manager if not a direct channel or if direct channel failed
	err := social.GetBotManager().SendMessage(platform, chatID, text)
	if err != nil {
		fmt.Printf("Butler: All send attempts failed for %s: %v\n", platform, err)
	}
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

	go b.executeTask(id, content, convID)

	return task, nil
}

func (b *Butler) executeTask(id, content string, convID string) {
	b.updateStatus(id, TaskStatusRunning, "")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	reply, err := b.QueryWithContext(ctx, content, "vibe")
	if err != nil {
		b.updateStatus(id, TaskStatusFailed, fmt.Sprintf("Error querying vibeauracle: %v", err))
		return
	}
	b.mu.Lock()
	if t, ok := b.tasks[id]; ok {
		t.Logs = append(t.Logs, reply)
	}
	b.mu.Unlock()
	b.updateStatus(id, TaskStatusCompleted, "Task completed successfully.")

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
	home, _ := os.UserHomeDir()
	
	// Use a non-blocking health check or one with a very short timeout
	client := vibe.NewClient()
	
	// Use a separate goroutine for the ping to avoid hanging the butler if the socket is stuck
	pingDone := make(chan error, 1)
	go func() {
		pingDone <- client.Ping()
	}()

	select {
	case err := <-pingDone:
		if err != nil {
			fmt.Printf("Butler: Vibeauracle socket unresponsive, attempting restart: %v\n", err)
			go b.restartVibeaura()
			return "System Health: Warning (Vibeauracle unresponsive). Self-healing initiated."
		}
	case <-time.After(2 * time.Second):
		fmt.Println("Butler: Health check ping timed out, attempting restart...")
		go b.restartVibeaura()
		return "System Health: Warning (Vibeauracle ping timeout). Self-healing initiated."
	}

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

func (b *Butler) restartVibeaura() {
	fmt.Println("Butler: Restarting vibeaura daemon...")
	// Try to stop it first just in case
	_ = exec.Command("vibeaura", "stop").Run()
	time.Sleep(1 * time.Second)
	
	// Start it
	cmd := exec.Command("vibeaura", "start")
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Butler: Failed to restart vibeaura: %v\n", err)
		return
	}
	
	// Wait a bit for it to initialize
	time.Sleep(3 * time.Second)
	fmt.Println("Butler: Vibeaura restart attempt completed.")
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
