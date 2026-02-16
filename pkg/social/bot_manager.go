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
	"time"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/memory"
)

type ContextualQuerier interface {
	QueryWithContext(ctx context.Context, prompt string, intent string) (string, error)
}

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
	Verbose  bool    `json:"verbose,omitempty"`

	// Social Affinity Metrics
	MTTR          time.Duration `json:"mttr,omitempty"`
	LastMessageAt time.Time     `json:"last_message_at,omitempty"`
	ReplyCount    int           `json:"reply_count,omitempty"`
}

type BotManager struct {
	bots      []BotConfig
	providers map[string]MessengerProvider
	mu        sync.RWMutex
	path      string

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
			path:      path,
			providers: make(map[string]MessengerProvider),
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

func (bm *BotManager) StartBots(ctx context.Context, history *memory.HistoryStore, querier ContextualQuerier, onTask func(platform, chatID, from, text string) string) {
	bm.mu.RLock()
	bots := make([]BotConfig, len(bm.bots))
	copy(bots, bm.bots)
	bm.mu.RUnlock()

	for i := range bots {
		go bm.runBot(ctx, &bots[i], history, querier, onTask)
	}
}

func (bm *BotManager) runBot(ctx context.Context, cfg *BotConfig, history *memory.HistoryStore, querier ContextualQuerier, onTask func(platform, chatID, from, text string) string) {
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

	bm.mu.Lock()
	bm.providers[cfg.Platform] = p
	bm.mu.Unlock()

	// Set commands
	commands := []BotCommand{
		{Text: "start", Description: "Start the bot and get the menu"},
		{Text: "mode", Description: "Switch agent operational mode"},
		{Text: "pay", Description: "Initiate experimental x402 payment"},
		{Text: "wallet", Description: "Check agent wallet balance (Base/Sepolia)"},
		{Text: "settle", Description: "Verify and settle pending intents"},
		{Text: "status", Description: "Check system and bot health"},
		{Text: "mission", Description: "Show current mission and deadline"},
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

			// Update MTTR if we were waiting for a reply
			if !cfg.LastMessageAt.IsZero() {
				latency := time.Since(cfg.LastMessageAt)
				bm.mu.Lock()
				cfg.MTTR = (cfg.MTTR*time.Duration(cfg.ReplyCount) + latency) / time.Duration(cfg.ReplyCount+1)
				cfg.ReplyCount++
				cfg.LastMessageAt = time.Time{} // Reset
				bm.mu.Unlock()
				bm.UpdateBot(*cfg)
			}

			// Handle Commands
			if bm.handleCommand(ctx, p, cfg, update, querier, onTask) {
				continue
			}

			// Handle Modes
			switch cfg.Mode {
			case ModeShell:
				bm.handleShellMode(ctx, p, cfg, text)
			case ModeAgent, ModeChat:
				bm.handleAgenticMode(ctx, p, cfg, text, history, querier, onTask)
			default:
				cfg.Mode = ModeChat
				bm.UpdateBot(*cfg)
				bm.handleAgenticMode(ctx, p, cfg, text, history, querier, onTask)
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

	bm.mu.Lock()
	cfg.LastMessageAt = time.Now()
	bm.mu.Unlock()
	bm.UpdateBot(*cfg)
}

func (bm *BotManager) handleCommand(ctx context.Context, p MessengerProvider, cfg *BotConfig, update Update, querier ContextualQuerier, onTask func(platform, chatID, from, text string) string) bool {
	text := update.Text

	if text == "/verbose" {
		cfg.Verbose = !cfg.Verbose
		bm.UpdateBot(*cfg)
		status := "OFF"
		if cfg.Verbose {
			status = "ON"
		}
		p.SendMessage(update.ChatID, fmt.Sprintf("📢 Verbose mode is now %s.", status), MessageOptions{})
		return true
	}

	if text == "/mode" || strings.HasPrefix(text, "/mode_") {
		bm.handleModeSwitch(p, cfg, update)
		return true
	}

	if text == "/pay" || strings.HasPrefix(text, "/pay_") || text == "/wallet" || strings.HasPrefix(text, "/wallet_") || text == "/settle" || strings.HasPrefix(text, "/settle_") {
		bm.handleSettlerCommand(ctx, p, cfg, update)
		return true
	}

	if !strings.HasPrefix(text, "/") && !strings.HasPrefix(text, "Mode:") {
		return false
	}

	if text == "/start" {
		bm.sendWelcome(p, update.ChatID, cfg)
		return true
	}

	// Route through the agentic loop even for commands
	// This allows the agent to challenge or mock the command request.
	p.SendAction(update.ChatID, ActionTyping)

	prompt := fmt.Sprintf("USER COMMAND: %s\n\nHandle this command. You can choose to execute it, ignore it, or challenge the user. Be punchy and mocking if you feel like it.", text)

	// Record in history first
	hist, _ := memory.NewHistoryStore()
	convID, _ := hist.GetOrCreateConversationForPlatform(cfg.Platform, cfg.OwnerID)
	_ = hist.AddMessage(convID, "user", text)

	go func() {
		// Use querier for context-aware query
		finalReply, err := querier.QueryWithContext(ctx, prompt, "agent")
		if err != nil {
			p.SendMessage(update.ChatID, "⚠️ Command system glitch. Don't touch anything.", MessageOptions{ParseMode: ParseModeHTML})
			return
		}

		p.SendMessage(update.ChatID, finalReply, MessageOptions{ParseMode: ParseModeHTML})
		hist, _ := memory.NewHistoryStore()
		_ = hist.AddMessage(convID, "assistant", finalReply)

		// Update MTTR since we sent a reply
		bm.mu.Lock()
		cfg.LastMessageAt = time.Now()
		bm.mu.Unlock()
		bm.UpdateBot(*cfg)
	}()

	return true
}

func (bm *BotManager) handleSettlerCommand(ctx context.Context, p MessengerProvider, cfg *BotConfig, update Update) {
	cmd := update.Text
	p.SendAction(update.ChatID, ActionTyping)

	// Simulation of SettlerEngine (Experimental Demo)
	// In production, this would dial UDS: ~/.config/settlerengine/settler.sock

	switch {
	case cmd == "/wallet" || cmd == "/wallet_info":
		resp := "💳 *SettlerEngine Wallet (Demo)*\n\n" +
			"*Address:* `0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199`\n" +
			"*Network:* Base Sepolia (84532)\n" +
			"*Balance:* `42.069 USDC`\n\n" +
			"_Status: Connected via UDS_"
		p.SendMessage(update.ChatID, resp, MessageOptions{ParseMode: ParseModeMarkdown})

	case cmd == "/pay":
		opts := MessageOptions{ParseMode: ParseModeMarkdown}
		if cfg.Platform == "telegram" {
			opts.Keyboard = NewPaymentKeyboard()
		}
		p.SendMessage(update.ChatID, "💸 *Experimental Payment Support*\nInitiate an x402 settlement via SettlerEngine:", opts)

	case strings.HasPrefix(cmd, "/pay_"):
		amount := "1.0"
		if cmd == "/pay_1_usdc" {
			amount = "1.0"
		}
		resp := fmt.Sprintf("💸 *Payment Initiated*\n\n"+
			"*Amount:* `%s USDC`\n"+
			"*Recipient:* `0xMerchant...`\n"+
			"*Protocol:* x402 Handshake\n\n"+
			"⏳ _Waiting for signature verification..._\n"+
			"✅ _SettlerEngine: Payment intent signed and broadcast._\n\n"+
			"*TX:* [0xabc123...](https://sepolia.basescan.org/tx/mock)", amount)
		p.SendMessage(update.ChatID, resp, MessageOptions{ParseMode: ParseModeMarkdown})

	case cmd == "/settle" || cmd == "/settle_all":
		resp := "🔄 *Settlement Engine*\n\n" +
			"Searching for pending x402 intents...\n" +
			"Found 1 pending settlement from `0xabc...`\n\n" +
			"✅ *Settled:* 0.5 USDC\n" +
			"Total processed: 0.5 USDC"
		p.SendMessage(update.ChatID, resp, MessageOptions{ParseMode: ParseModeMarkdown})
	}
}

func (bm *BotManager) handleModeSwitch(p MessengerProvider, cfg *BotConfig, update Update) {
	text := update.Text
	newMode := ""

	// Handle inline callback data
	switch text {
	case "/mode_chat":
		newMode = string(ModeChat)
	case "/mode_agent":
		newMode = string(ModeAgent)
	case "/mode_shell":
		newMode = string(ModeShell)
	}

	// Handle text-based mode switch
	if newMode == "" {
		if strings.Contains(text, "Chat") {
			newMode = string(ModeChat)
		} else if strings.Contains(text, "Agent") {
			newMode = string(ModeAgent)
		} else if strings.Contains(text, "Shell") {
			newMode = string(ModeShell)
		}
	}

	if text == "/mode" {
		opts := MessageOptions{ParseMode: ParseModeMarkdown}
		if cfg.Platform == "telegram" {
			opts.Keyboard = NewModeInlineKeyboard()
		}
		p.SendMessage(update.ChatID, "⚙️ *Operational Mode*\nSelect the agent's current focus:", opts)
		return
	}

	if newMode != "" {
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

	bm.mu.Lock()
	cfg.LastMessageAt = time.Now()
	bm.mu.Unlock()
	bm.UpdateBot(*cfg)
}

func (bm *BotManager) handleAgenticMode(ctx context.Context, p MessengerProvider, cfg *BotConfig, text string, history *memory.HistoryStore, querier ContextualQuerier, onTask func(platform, chatID, from, text string) string) {
	p.SendAction(cfg.OwnerID, ActionTyping)

	// Use the task handler which manages the butler state
	reply := onTask(cfg.Platform, cfg.OwnerID, cfg.OwnerID, text)
	if reply != "" {
		p.SendMessage(cfg.OwnerID, reply, MessageOptions{ParseMode: ParseModeHTML})
	}

	bm.mu.Lock()
	cfg.LastMessageAt = time.Now()
	bm.mu.Unlock()
	bm.UpdateBot(*cfg)
}

func (bm *BotManager) SendMessage(platform string, chatID string, text string) error {
	bm.mu.RLock()
	p, ok := bm.providers[platform]
	bm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("provider for platform %s not found or not active", platform)
	}

	return p.SendMessage(chatID, text, MessageOptions{ParseMode: ParseModeHTML})
}

func (bm *BotManager) BroadcastLog(text string) {
	fmt.Printf("Butler [Log]: %s\n", text)
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	for _, cfg := range bm.bots {
		if cfg.Verbose && cfg.OwnerID != "" {
			if p, ok := bm.providers[cfg.Platform]; ok {
				_ = p.SendMessage(cfg.OwnerID, "📝 [Log] "+text, MessageOptions{ParseMode: ParseModeHTML})
			}
		}
	}
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
