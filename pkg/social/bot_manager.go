package social

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/memory"
	"github.com/nathfavour/auracrab/pkg/vibe"
)

type BotMode string

const (
	ModeChat  BotMode = "chat"
	ModeAgent BotMode = "agent"
	ModeShell BotMode = "shell"
)

type BotConfig struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Token    string  `json:"token"`
	Platform string  `json:"platform"` // "telegram", "discord"
	OwnerID  string  `json:"owner_id,omitempty"`
	Mode     BotMode `json:"mode,omitempty"`
}

type BotManager struct {
	bots []BotConfig
	mu   sync.RWMutex
	path string

	// Shell blacklist from POC
	shellBlacklist []string
}

var (
	botManagerInstance *BotManager
	botOnce            sync.Once
)

func GetBotManager() *BotManager {
	botOnce.Do(func() {
		path := filepath.Join(config.DataDir(), "bots.json")
		botManagerInstance = &BotManager{
			path: path,
			shellBlacklist: []string{
				"rm ", "mkfs", "dd ", "fdisk", "reboot", "shutdown", "init ",
				"chmod", "chown", "mv /", "> /dev", "kill", "halt", "poweroff",
			},
		}
		botManagerInstance.load()
	})
	return botManagerInstance
}

func (bm *BotManager) load() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	data, err := os.ReadFile(bm.path)
	if err != nil {
		bm.bots = []BotConfig{}
		return
	}

	if err := json.Unmarshal(data, &bm.bots); err != nil {
		log.Printf("Error loading bots: %v", err)
		bm.bots = []BotConfig{}
	}
}

func (bm *BotManager) save() error {
	data, err := json.MarshalIndent(bm.bots, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(bm.path, data, 0600)
}

func (bm *BotManager) AddBot(cfg BotConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if cfg.Mode == "" {
		cfg.Mode = ModeChat
	}
	bm.bots = append(bm.bots, cfg)
	return bm.save()
}

func (bm *BotManager) ListBots() []BotConfig {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bots
}

func (bm *BotManager) UpdateBot(cfg BotConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for i, b := range bm.bots {
		if b.Token == cfg.Token || (b.ID != "" && b.ID == cfg.ID) {
			bm.bots[i] = cfg
			return bm.save()
		}
	}
	return fmt.Errorf("bot not found")
}

func (bm *BotManager) StartBots(ctx context.Context, history *memory.HistoryStore, onTask func(from, text string) string) {
	bm.mu.RLock()
	bots := make([]BotConfig, len(bm.bots))
	copy(bots, bm.bots)
	bm.mu.RUnlock()

	for i := range bots {
		go bm.runBot(ctx, &bots[i], history, onTask)
	}
}

func (bm *BotManager) runBot(ctx context.Context, cfg *BotConfig, history *memory.HistoryStore, onTask func(from, text string) string) {
	var p MessengerProvider
	var err error

	switch cfg.Platform {
	case "telegram":
		p, err = NewTelegramProvider(cfg.Token)
	case "discord":
		p, err = NewDiscordProvider(cfg.Token)
	default:
		log.Printf("Unsupported platform: %s", cfg.Platform)
		return
	}

	if err != nil {
		log.Printf("Failed to start bot %s: %v", cfg.Name, err)
		return
	}

	// Set commands
	commands := []BotCommand{
		{Text: "start", Description: "Start the bot and get the menu"},
		{Text: "mode", Description: "Switch operation mode"},
		{Text: "status", Description: "Check system and bot status"},
		{Text: "help", Description: "Show help information"},
	}
	p.SetCommands(commands)

	updates, err := p.GetUpdates(ctx)
	if err != nil {
		log.Printf("Bot %s updates error: %v", cfg.Name, err)
		return
	}

	log.Printf("Bot %s (%s) started in %s mode", cfg.Name, cfg.Platform, cfg.Mode)

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}

			// Registration / Owner Check
			if cfg.OwnerID == "" {
				cfg.OwnerID = update.ChatID
				bm.UpdateBot(*cfg)
				bm.sendWelcome(p, update.ChatID, cfg)
				continue
			}

			if update.ChatID != cfg.OwnerID {
				p.SendMessage(update.ChatID, "Unauthorized access denied.", MessageOptions{})
				continue
			}

			text := update.Text
			if text == "" {
				continue
			}

			// Handle Commands
			if bm.handleCommand(ctx, p, cfg, update, onTask) {
				continue
			}

			// Handle Modes
			switch cfg.Mode {
			case ModeShell:
				bm.handleShellMode(ctx, p, cfg, text)
			case ModeAgent, ModeChat:
				bm.handleAgenticMode(ctx, p, cfg, text, history, onTask)
			default:
				cfg.Mode = ModeChat
				bm.UpdateBot(*cfg)
				bm.handleAgenticMode(ctx, p, cfg, text, history, onTask)
			}
		}
	}
}

func (bm *BotManager) sendWelcome(p MessengerProvider, chatID string, cfg *BotConfig) {
	opts := MessageOptions{}
	if cfg.Platform == "telegram" {
		opts.Keyboard = TelegramModeKeyboard
	}
	p.SendMessage(chatID, "Welcome, Boss. I am your Auracrab Gateway. I've registered you as my owner.", opts)
}

func (bm *BotManager) handleCommand(ctx context.Context, p MessengerProvider, cfg *BotConfig, update Update, onTask func(from, text string) string) bool {
	text := update.Text
	chatID := update.ChatID

	switch {
	case text == "/start":
		bm.sendWelcome(p, chatID, cfg)
		return true
	case text == "/status":
		p.SendAction(chatID, ActionTyping)
		reply := onTask(fmt.Sprintf("%v", update.RawFrom), "get_status_internal")
		p.SendMessage(chatID, "📊 *System Status*\n"+reply, MessageOptions{ParseMode: ParseModeHTML})
		return true
	case text == "/help":
		help := `🚀 *Auracrab Gateway Help*\n\n*Modes:*\n• *Chat:* Conversational AI focus.\n• *Agent:* Full agentic power (tool use).\n• *Shell:* Direct bash access (restricted).\n\n*Commands:*\n/mode - Switch modes\n/status - Check health\n/start - Refresh menu\n\n_Safety: Shell mode is direct. Be careful._`
		p.SendMessage(chatID, help, MessageOptions{ParseMode: ParseModeHTML})
		return true
	case strings.HasPrefix(text, "/mode") || strings.HasPrefix(text, "Mode:"):
		bm.handleModeSwitch(p, cfg, text)
		return true
	}
	return false
}

func (bm *BotManager) handleModeSwitch(p MessengerProvider, cfg *BotConfig, text string) {
	newMode := ""
	if strings.Contains(text, "Chat") {
		newMode = string(ModeChat)
	} else if strings.Contains(text, "Agent") {
		newMode = string(ModeAgent)
	} else if strings.Contains(text, "Shell") {
		newMode = string(ModeShell)
	} else {
		parts := strings.Split(text, " ")
		if len(parts) > 1 {
			newMode = strings.ToLower(parts[1])
		}
	}

	if newMode == "chat" || newMode == "agent" || newMode == "shell" {
		cfg.Mode = BotMode(newMode)
		bm.UpdateBot(*cfg)
		opts := MessageOptions{}
		if cfg.Platform == "telegram" {
			opts.Keyboard = TelegramModeKeyboard
		}
		p.SendMessage(cfg.OwnerID, fmt.Sprintf("✅ Mode switched to: %s", strings.ToUpper(newMode)), opts)
	}
}

func (bm *BotManager) handleShellMode(ctx context.Context, p MessengerProvider, cfg *BotConfig, command string) {
	lowerCmd := strings.ToLower(command)
	for _, restricted := range bm.shellBlacklist {
		if strings.Contains(lowerCmd, restricted) {
			p.SendMessage(cfg.OwnerID, "🛑 *SECURITY ALERT*: Command is restricted.", MessageOptions{ParseMode: ParseModeHTML})
			return
		}
	}

	p.SendAction(cfg.OwnerID, ActionTyping)
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	output, err := cmd.CombinedOutput()

	var resp string
	if err != nil {
		resp = fmt.Sprintf("❌ *Error:* %v\n\n```\n%s\n```", err, string(output))
	} else {
		if len(output) == 0 {
			resp = "✅ Executed."
		} else {
			resp = fmt.Sprintf("```\n%s\n```", string(output))
		}
	}
	p.SendMessage(cfg.OwnerID, resp, MessageOptions{ParseMode: ParseModeHTML})
}

func (bm *BotManager) handleAgenticMode(ctx context.Context, p MessengerProvider, cfg *BotConfig, text string, history *memory.HistoryStore, onTask func(from, text string) string) {
	p.SendAction(cfg.OwnerID, ActionTyping)

	// Use history
	convID, _ := history.GetOrCreateConversationForPlatform(cfg.Platform, cfg.OwnerID)
	
	prompt := text
	intent := "crud"
	if cfg.Mode == ModeChat {
		prompt = "CONVERSATIONAL MODE: Concise response. Minimal tools.\n\n" + text
		intent = "ask"
	} else {
		prompt = "AGENT MODE: Use tools to solve.\n\n" + text
	}

	go func() {
		// Run through vibeaura UDS
		client := vibe.NewClient()
		finalReply, err := client.Query(prompt, intent)
		if err != nil {
			p.SendMessage(cfg.OwnerID, fmt.Sprintf("⚠️ *Error*\n```\n%v\n```", err), MessageOptions{ParseMode: ParseModeHTML})
			return
		}

		if finalReply == "" {
			finalReply = "_No response._"
		}

		p.SendMessage(cfg.OwnerID, finalReply, MessageOptions{ParseMode: ParseModeHTML})
		
		// Record in history
		_ = history.AddMessage(convID, "user", text)
		_ = history.AddMessage(convID, "assistant", finalReply)
	}()
}

// Utility functions
func EscapeHTML(text string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	).Replace(text)
}

func StripANSI(str string) string {
	const ansi = `[\x1b\x9b][[()#;?]*(?:[a-zA-Z\d]*(?:;[-a-zA-Z\d/#&.:=?%@~]*)*)?[0-9A-ORZcf-nqry=><]`
	re := regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}