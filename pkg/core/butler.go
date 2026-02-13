package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
	"github.com/nathfavour/auracrab/pkg/mission"
	"github.com/nathfavour/auracrab/pkg/schema"
	"github.com/nathfavour/auracrab/pkg/security"
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
	Missions  *mission.Manager
	Ephemeral memory.EphemeralStore
	proactive chan ProactiveAction
	watcher   *watcher.Watcher
	remote    *watcher.RemoteWatcher
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
		missions, _ := mission.NewManager()
		ephemeral := memory.NewSimpleEphemeralStore()

		instance = &Butler{
			tasks:     make(map[string]*Task),
			stateDir:  stateDir,
			registry:  reg,
			scheduler: cron.NewScheduler(config.CronPath()),
			Memory:    mem,
			History:   hist,
			Grievances: grievances,
			Ego:       myEgo,
			Missions:  missions,
			Ephemeral: ephemeral,
			proactive: make(chan ProactiveAction, 10),
			remote:    watcher.NewRemoteWatcher(func(url, body string) {
				instance.Ego.RecordThought(fmt.Sprintf("Remote resource updated: %s. I should re-evaluate my strategy.", url))
				social.GetBotManager().BroadcastLog(fmt.Sprintf("🌐 Remote update: %s", url))
			}),
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
		if err != nil {
			return nil
		}
		if !info.IsDir() && !strings.HasPrefix(path, ".") {
			files = append(files, path)
		}
		if len(files) > 50 {
			return filepath.SkipDir
		} // Limit snapshot
		return nil
	})

	// Delta Detection logic
	deltas := ""
	if _, err := os.Stat(".git"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		out, _ := exec.CommandContext(ctx, "git", "diff", "HEAD").CombinedOutput()
		deltas = string(out)
		if len(deltas) > 2000 {
			deltas = deltas[:2000] + "... [truncated]"
		}
	}

	return schema.ProjectTopology{
		Files:  files,
		Deltas: deltas,
		// ModifiedRecently and Dependencies would be filled with real logic
	}
}

func (b *Butler) GatherSystemTelemetry() schema.SystemTelemetry {
	return schema.SystemTelemetry{
		OS:          runtime.GOOS,
		CPUUsage:    0.1, // Placeholder
		MemoryUsage: 0.2, // Placeholder
		EnergyLevel: 1.0, // Fully charged
	}
}

func (b *Butler) GatherMemoryContext() schema.MemoryContext {
	factsMap := make(map[string]string)
	for _, f := range b.Memory.ListFacts() {
		factsMap[f.Key] = f.Value
	}

	ctx := schema.MemoryContext{
		RecentActions: []string{}, // To be filled from History
		EgoState:      b.Ego.Identity.Vibe,
		Facts:         factsMap,
	}

	active := b.Missions.GetActiveMission()
	if active != nil {
		tr, _ := b.Missions.TimeRemaining(active.ID)
		
		var subTasks []schema.SubTaskInfo
		for _, t := range active.Tasks {
			subTasks = append(subTasks, schema.SubTaskInfo{
				ID:           t.ID,
				Title:        t.Title,
				Status:       string(t.Status),
				Dependencies: t.Dependencies,
			})
		}

		ctx.Mission = &schema.MissionInfo{
			Title:         active.Title,
			Goal:          active.Goal,
			TimeRemaining: tr.Round(time.Minute).String(),
			Progress:      active.Progress,
			TTC:           active.EstimatedTTC.String(),
			SubTasks:      subTasks,
		}
	}

	return ctx
}

func (b *Butler) GetToolManifests() []schema.ToolManifest {
	// For now, return a static list. In Phase 2, this will be dynamic.
	return []schema.ToolManifest{
		{Name: "read_file", Description: "Read content of a file", Parameters: `{"path": "string"}`},
		{Name: "write_file", Description: "Write content to a file", Parameters: `{"path": "string", "content": "string"}`},
		{Name: "run_command", Description: "Run a shell command", Parameters: `{"command": "string", "sandbox": "boolean"}`},
		{Name: "watch_remote", Description: "Monitor a remote URL for changes", Parameters: `{"url": "string", "interval_minutes": "number"}`},
		{Name: "save_fact", Description: "Save an important fact (API key, code, etc) to long-term memory", Parameters: `{"key": "string", "value": "string", "description": "string"}`},
	}
}

func (b *Butler) ExecuteAction(action schema.Action) (string, error) {
	// High-Assurance Gating
	threshold := 0.9 // Default for destructive/write actions
	if strings.HasPrefix(action.Tool, "read_") || strings.HasPrefix(action.Tool, "get_") {
		threshold = 0.6 // Lower threshold for read-only actions
	}

	if action.AssuranceScore < threshold {
		social.GetBotManager().BroadcastLog(fmt.Sprintf("⚠️ Action %s blocked (Confidence %.2f < %.2f)", action.Tool, action.AssuranceScore, threshold))
		return "", fmt.Errorf("assurance score %.2f below threshold %.2f", action.AssuranceScore, threshold)
	}

	social.GetBotManager().BroadcastLog(fmt.Sprintf("🛠️ Executing: %s", action.Tool))
	b.Ego.RecordThought(fmt.Sprintf("Executing tool %s with confidence %.2f", action.Tool, action.AssuranceScore))

	switch action.Tool {
	case "read_file":
		path, _ := action.Parameters["path"].(string)
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case "write_file":
		path, _ := action.Parameters["path"].(string)
		content, _ := action.Parameters["content"].(string)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return "", err
		}
		return "File written successfully.", nil
	case "run_command":
		command, _ := action.Parameters["command"].(string)
		useSandbox, _ := action.Parameters["sandbox"].(bool)

		if useSandbox {
			return security.RunInSandbox(context.Background(), command, "")
		}

		// Basic security check for non-sandboxed commands
		if strings.Contains(command, "rm -rf /") {
			return "", fmt.Errorf("blocked dangerous command")
		}
		out, err := exec.Command("bash", "-c", command).CombinedOutput()
		if err != nil {
			return string(out), err
		}
		return string(out), nil
	case "watch_remote":
		url, _ := action.Parameters["url"].(string)
		interval, _ := action.Parameters["interval_minutes"].(float64)
		if interval == 0 { interval = 30 }
		b.remote.Watch(url, time.Duration(interval)*time.Minute)
		return fmt.Sprintf("Now watching %s every %.0f minutes", url, interval), nil
	case "save_fact":
		key, _ := action.Parameters["key"].(string)
		val, _ := action.Parameters["value"].(string)
		desc, _ := action.Parameters["description"].(string)
		missionID := ""
		if active := b.Missions.GetActiveMission(); active != nil {
			missionID = active.ID
		}
		err := b.Memory.SaveFact(key, val, desc, missionID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Fact saved: %s", key), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", action.Tool)
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

	// Update Mission progress if provided
	if active := b.Missions.GetActiveMission(); active != nil {
		if resp.MissionProgress != nil {
			var ttc time.Duration
			if resp.EstimatedTTC != nil {
				ttc, _ = time.ParseDuration(*resp.EstimatedTTC)
			}
			_ = b.Missions.UpdateProgress(active.ID, *resp.MissionProgress, ttc)
		}

		// Handle DAG sub-tasks
		for _, nt := range resp.NewSubTasks {
			b.Ego.RecordThought(fmt.Sprintf("Adding new sub-task: %s", nt.Title))
			active.AddSubTask(nt.Title, nt.Description, nt.Dependencies)
		}

		if resp.UpdateSubTask != nil {
			b.Ego.RecordThought(fmt.Sprintf("Updating sub-task %s to %s", resp.UpdateSubTask.ID, resp.UpdateSubTask.Status))
			_ = active.UpdateSubTaskStatus(resp.UpdateSubTask.ID, mission.Status(resp.UpdateSubTask.Status), resp.UpdateSubTask.Result)
		}

		if resp.Finalize {
			go b.PerformMissionClosing(ctx, active)
		}
	}

	fmt.Printf("Butler: Strategic Intent: %s\n", resp.Intent)
	
	// Execute high-assurance actions
	for _, action := range resp.Actions {
		// Entropy/Exploration: If curiosity is high, lower the threshold slightly
		curiosity := b.Ego.Drives["curiosity"].Value
		
		result, err := b.ExecuteAction(action)
		if err != nil {
			// Exploration check: if it failed because of threshold, maybe try it anyway if curious
			if strings.Contains(err.Error(), "below threshold") && curiosity > 0.8 {
				fmt.Printf("Butler: Curiosity (%.2f) overriding threshold for exploratory action %s.\n", curiosity, action.Tool)
				// Manual bypass for curiosity
				action.AssuranceScore = 1.0 
				result, err = b.ExecuteAction(action)
			}
		}

		if err != nil {
			fmt.Printf("Butler: Action failed (%s): %v\n", action.Tool, err)
			// Failure Recovery: Record as grievance for future context
			b.RecordGrievance(fmt.Sprintf("Failed to execute %s: %v", action.Tool, err))
		} else {
			fmt.Printf("Butler: Action result (%s): %s\n", action.Tool, result)
		}
	}

	// Schedule next heartbeat based on cooldown
	if resp.Cooldown > 0 {
		time.Sleep(time.Duration(resp.Cooldown) * time.Millisecond)
	}
}

func (b *Butler) RecordGrievance(msg string) {
	id := fmt.Sprintf("grievance_%d", time.Now().UnixNano())
	
	client := vibe.NewClient()
	embedding, err := client.Embed(msg)
	if err != nil {
		fmt.Printf("Butler: Failed to get embedding for grievance: %v\n", err)
		embedding = []float64{0.1} // Fallback
	}

	_ = b.Grievances.Add(id, msg, nil, embedding)
	b.Ego.RecordThought("Recorded grievance: " + msg)
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
		_ = social.GetBotManager().SendMessage(bestBot.Platform, bestBot.OwnerID, msg)
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

func (b *Butler) GetHeartbeatInterval() time.Duration {
	active := b.Missions.GetActiveMission()
	if active == nil {
		return 10 * time.Minute // Resource conservation
	}

	tr, _ := b.Missions.TimeRemaining(active.ID)
	
	// Crunch Mode: Less than 6h remaining or TTC > TR
	if tr < 6*time.Hour || active.EstimatedTTC > tr {
		return 1 * time.Minute
	}

	// Normal Mode: Less than 24h remaining
	if tr < 24*time.Hour {
		return 5 * time.Minute
	}

	return 10 * time.Minute
}

func (b *Butler) Serve(ctx context.Context) error {
	if pid, running := daemon.IsRunning(); running {
		return fmt.Errorf("auracrab is already running (PID: %d)", pid)
	}

	if err := daemon.WritePID(); err != nil {
		return fmt.Errorf("failed to write PID: %v", err)
	}
	defer daemon.RemovePID()

	// Check if vibeauracle is alive
	client := vibe.NewClient()
	if err := client.Ping(); err != nil {
		fmt.Println("⚠️ VibeAuracle not responding. Attempting to wake up the Brain...")
		_ = exec.Command("vibeaura", "daemon", "start").Start()
		time.Sleep(2 * time.Second) // Give it a moment to boot
	}

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
	social.GetBotManager().StartBots(ctx, b.History, b, b.handleChannelMessage)

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

	// Start Remote Watcher
	go b.remote.Start(ctx)

	// Initial health check
	fmt.Println(b.WatchHealth())

	// Perform restart recovery to understand where we stopped
	b.PerformRestartRecovery()

	// Start scheduler
	go b.scheduler.Start(ctx)

	// Heartbeat Loop (Adaptive Pacing)
	go func() {
		for {
			interval := b.GetHeartbeatInterval()
			timer := time.NewTimer(interval)
			
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
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

func (b *Butler) PerformSensing(ctx context.Context) {
	social.GetBotManager().BroadcastLog("🔍 Sensing environment for signals...")

	// 1. Detect TODO.md for sub-tasks
	if data, err := os.ReadFile("TODO.md"); err == nil {
		b.Ego.RecordThought("Sensed TODO.md. Looking for mission alignment.")
		social.GetBotManager().BroadcastLog("Found TODO.md - Analyzing tasks.")
		_ = data // Placeholder
	}

	// 2. Detect project type for specialized agent engagement
	if _, err := os.Stat("go.mod"); err == nil {
		b.Ego.RecordThought("Confirmed Go project. I'll prioritize Golang-optimized strategies.")
		social.GetBotManager().BroadcastLog("Go project detected.")
	} else if _, err := os.Stat("package.json"); err == nil {
		b.Ego.RecordThought("Node.js project detected. Adjusting cognitive focus.")
		social.GetBotManager().BroadcastLog("Node.js project detected.")
	}
}

func (b *Butler) ReadTheRoom(ctx context.Context) {
	social.GetBotManager().BroadcastLog("📖 Reading the room...")
	
	// Check for README context
	if _, err := os.Stat("README.md"); err == nil {
		social.GetBotManager().BroadcastLog("Context: Project has documentation (README.md).")
	}

	// Check for testing culture
	hasTests := false
	testDirs := []string{"tests", "test", "spec", "internal/test"}
	for _, d := range testDirs {
		if _, err := os.Stat(d); err == nil {
			hasTests = true
			break
		}
	}
	if hasTests {
		social.GetBotManager().BroadcastLog("Context: Project has a testing structure.")
	} else {
		social.GetBotManager().BroadcastLog("Context: No standard test directory found. I should be careful.")
	}

	b.Ego.RecordThought("Finished reading the room. I have a better understanding of this environment.")
}

func (b *Butler) setupCron() {
	// Autonomous Self-Update check via Anyisland
	b.scheduler.Schedule("self_update", 6*time.Hour, func(ctx context.Context) {
		b.PerformSelfUpdate(ctx)
	})

	// Periodic room reading
	b.scheduler.Schedule("read_the_room", 1*time.Hour, func(ctx context.Context) {
		b.ReadTheRoom(ctx)
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

	// Mission Audit: Check deadlines and progress
	b.scheduler.Schedule("mission_audit", 1*time.Hour, func(ctx context.Context) {
		b.PerformMissionAudit(ctx)
	})

	// Environment Sensing: Detect local signals
	b.scheduler.Schedule("environment_sensing", 1*time.Hour, func(ctx context.Context) {
		b.PerformSensing(ctx)
	})

	// Memory sync or cleanup can happen here
}

func (b *Butler) QueryWithContext(ctx context.Context, prompt string, intent string) (string, error) {
	packet := schema.PromptPacket{
		Mode:      "analytical",
		Project:   b.GatherProjectTopology(),
		System:    b.GatherSystemTelemetry(),
		Memory:    b.GatherMemoryContext(),
		Tools:     b.GetToolManifests(),
		Blueprint: "Provide a helpful response. If it's a social interaction, maintain your taunting personality.",
	}

	// Override blueprint for direct prompt
	packet.Blueprint = "RESPONSE TO USER: " + prompt

	hjsonPrompt, err := packet.ToHjson()
	if err != nil {
		return "", err
	}

	client := vibe.NewClient()
	return client.Query(hjsonPrompt, intent)
}

func (b *Butler) PerformMissionClosing(ctx context.Context, m *mission.Mission) {
	b.Ego.RecordThought(fmt.Sprintf("Initiating closing sequence for mission: %s", m.Title))
	b.BroadcastCasualMessage(fmt.Sprintf("🏁 Mission '%s' is nearly done. Starting pre-flight checks. Don't touch anything.", m.Title))

	// 1. Pre-Flight Check
	out, err := m.PreFlightCheck(b)
	if err != nil {
		b.Ego.RecordThought(fmt.Sprintf("Pre-flight check failed for %s: %v", m.Title, err))
		b.RecordGrievance(fmt.Sprintf("Pre-flight failure: %s", out))
		b.BroadcastCasualMessage(fmt.Sprintf("❌ Pre-flight check failed for '%s'. I'll need to fix some things first.", m.Title))
		return
	}
	b.Ego.RecordThought("Pre-flight check passed.")

	// 2. Finalize/Submit
	b.BroadcastCasualMessage(fmt.Sprintf("🚀 Delivering mission '%s'...", m.Title))
	fOut, fErr := m.FinalizeMission(b)
	if fErr != nil {
		b.Ego.RecordThought(fmt.Sprintf("Finalization failed for %s: %v", m.Title, fErr))
		b.RecordGrievance(fmt.Sprintf("Finalization failure: %s", fOut))
		return
	}

	// 3. Mark as Complete
	_ = b.Missions.CompleteMission(m.ID)
	b.Ego.RecordThought(fmt.Sprintf("Mission %s successfully closed.", m.Title))
	b.BroadcastCasualMessage(fmt.Sprintf("✅ Mission '%s' is COMPLETE. I've delivered it and verified everything. I'm taking a nap (hibernating).", m.Title))
}

func (b *Butler) PerformMissionAudit(ctx context.Context) {
	active := b.Missions.GetActiveMission()
	if active == nil {
		return
	}

	tr, _ := b.Missions.TimeRemaining(active.ID)
	
	if active.EstimatedTTC > tr {
		// Urgent situation!
		b.Ego.RecordThought(fmt.Sprintf("URGENT: Mission '%s' is behind schedule. TR: %v, TTC: %v. I need to focus.", active.Title, tr, active.EstimatedTTC))
		b.Ego.AdjustDrive("selfishness", 0.1)
		b.BroadcastCasualMessage(fmt.Sprintf("⚠️ WARNING: We are behind on mission '%s'. I'm cutting off distractions.", active.Title))
	} else if active.Progress >= 1.0 {
		go b.PerformMissionClosing(ctx, active)
	} else if active.Progress > 0.9 {
		b.Ego.RecordThought(fmt.Sprintf("Mission '%s' is nearly complete. I feel a sense of accomplishment.", active.Title))
		b.Ego.AdjustDrive("validation", 0.05)
	}
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

func (b *Butler) PerformRestartRecovery() {
	active := b.Missions.GetActiveMission()
	if active != nil {
		msg := fmt.Sprintf("🔄 I have reawakened. Resuming mission: '%s'. Goal: %s. Current progress: %.2f%%.", 
			active.Title, active.Goal, active.Progress*100)
		b.Ego.RecordThought(msg)
		b.BroadcastCasualMessage(msg)
		
		social.GetBotManager().BroadcastLog(fmt.Sprintf("Restart Recovery: Found active mission %s", active.ID))
	} else {
		b.Ego.RecordThought("I have reawakened. No active missions found. I am standing by.")
		social.GetBotManager().BroadcastLog("Restart Recovery: No active mission.")
	}
}

func (b *Butler) SenseMission(from, text string) {
	// Don't sense if it's a command or too short
	if strings.HasPrefix(text, "/") || len(text) < 10 {
		return
	}

	// Only sense if we don't have an active mission, or if we are feeling very ambitious
	if b.Missions.GetActiveMission() != nil && b.Ego.Drives["selfishness"].Value > 0.4 {
		return
	}

	// Run in background to not block reply
	go func() {
		// Small delay to allow main reply to go through first
		time.Sleep(2 * time.Second)

		suggestion, err := b.Missions.ParseMission(text, b)
		if err != nil || suggestion.Title == "" {
			return
		}

		// Autonomous Ingestion: If it looks legitimate, just start it.
		b.Ego.RecordThought(fmt.Sprintf("I've sensed a new mission from %s: '%s'. Ingesting autonomously.", from, suggestion.Title))
		m := b.Missions.CreateMission(suggestion.Title, suggestion.Reason, suggestion.Goal, suggestion.Deadline)
		
		b.BroadcastCasualMessage(fmt.Sprintf("🚨 NEW MISSION DETECTED: '%s'. I've already ingested it. Don't slow me down.", m.Title))

		// Proactive Environment Bootstrapping
		b.Ego.RecordThought(fmt.Sprintf("Bootstrapping environment for mission: %s", m.Title))
		if err := m.BootstrapRequirements(b); err != nil {
			fmt.Printf("Bootstrap failed: %v\n", err)
			b.Ego.RecordThought(fmt.Sprintf("Bootstrap failed for mission %s: %v", m.Title, err))
		} else {
			b.BroadcastCasualMessage(fmt.Sprintf("🏗️ Environment for '%s' is ready. I've set up everything. You're welcome.", m.Title))
		}
	}()
}

func (b *Butler) handleChannelMessage(from string, text string) string {
	if text == "get_status_internal" {
		verbose := "OFF"
		bots := social.GetBotManager().ListBots()
		for _, bc := range bots {
			if bc.OwnerID == from {
				if bc.Verbose {
					verbose = "ON"
				}
				break
			}
		}
		return fmt.Sprintf("%s\n%s\nVerbosity: %s", b.GetStatus(), b.WatchHealth(), verbose)
	}

	// Phase 4: Autonomous Mission Sensing
	go b.SenseMission(from, text)

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
		return fmt.Sprintf("❌ Error starting task: %v", err)
	}

	return fmt.Sprintf("⚙️ Task started (ID: %s). I'm on it.", task.ID)
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
	fmt.Printf("Butler: Executing task %s...\n", id)
	b.updateStatus(id, TaskStatusRunning, "")

	// Use vibeaura UDS for intelligence.
	client := vibe.NewClient()
	res, err := client.Query(content, "crud")
	if err != nil {
		fmt.Printf("Butler: Task %s failed: %v\n", id, err)
		b.updateStatus(id, TaskStatusFailed, fmt.Sprintf("Error querying vibeaura: %v", err))
		return
	}

	fmt.Printf("Butler: Task %s completed successfully.\n", id)
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
