package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nathfavour/auracrab/pkg/memory"
	"github.com/nathfavour/auracrab/pkg/vault"
)

// TelegramChannel is a real Telegram integration using long-polling.
type TelegramChannel struct {
	Token    string
	offset   int
	stateDir string
	history  *memory.HistoryStore
	bot      *tgbotapi.BotAPI
}

func (t *TelegramChannel) Name() string {
	return "telegram"
}

func (t *TelegramChannel) Start(ctx context.Context, onMessage func(platform string, chatID string, from string, text string) string) error {
	home, _ := os.UserHomeDir()
	t.stateDir = filepath.Join(home, ".local", "share", "auracrab", "telegram")
	_ = os.MkdirAll(t.stateDir, 0755)

	t.loadOffset()

	// Initialize history store reference
	var err error
	t.history, err = memory.NewHistoryStore()
	if err != nil {
		log.Printf("Warning: Telegram could not initialize history store: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(t.Token)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %v", err)
	}
	t.bot = bot

	log.Printf("Auracrab: Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(t.offset + 1)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updates:
				if update.Message == nil {
					continue
				}

				// Update offset
				if update.UpdateID > t.offset {
					t.offset = update.UpdateID
					t.saveOffset()
				}

				from := fmt.Sprintf("@%s", update.Message.From.UserName)
				if update.Message.From.UserName == "" {
					from = fmt.Sprintf("%d", update.Message.From.ID)
				}

				text := update.Message.Text
				if text == "" {
					continue
				}

				chatID := update.Message.Chat.ID
				chatIDStr := fmt.Sprintf("%d", chatID)
				v := vault.GetVault()

				// Check Database authorization first
				isAllowed := false
				if t.history != nil {
					isAllowed, _ = t.history.IsAuthorized("telegram", chatIDStr)
				}

				// Fallback to Vault whitelist if not explicitly in DB
				if !isAllowed {
					allowedChats, _ := v.Get("TELEGRAM_ALLOWED_CHATS")
					if allowedChats != "" {
						for _, idStr := range strings.Split(allowedChats, ",") {
							if strings.TrimSpace(idStr) == chatIDStr {
								isAllowed = true
								// Persist to DB for faster lookup next time
								if t.history != nil {
									_ = t.history.AuthorizeEntity("telegram", chatIDStr)
								}
								break
							}
						}
					} else {
						// If nothing configured at all, we might want to be strict
						// but for now, let's keep the user's "allowed to message" requirement.
						// We'll default to false if any TELEGRAM_ALLOWED_CHATS is set,
						// or true if it's completely fresh (to avoid locking users out immediately).
						// But with the "no DM first" reminder, let's be more careful.
						isAllowed = false
					}
				}

				log.Printf("[%s] (Chat: %d) %s", from, chatID, text)

				// Handle internal bot commands first
				if strings.HasPrefix(text, "/") {
					cmd := strings.Split(text, " ")[0]
					switch cmd {
					case "/id":
						msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Your Chat ID is: %d\nTo allow this chat, run `/config set TELEGRAM_ALLOWED_CHATS %d` in the Auracrab TUI.", chatID, chatID))
						bot.Send(msg)
						continue
					case "/start":
						msg := tgbotapi.NewMessage(chatID, "🦀 *Auracrab Telegram Bot*\n\nI am your autonomous agent. I follow a strict 'No DM first' policy. Please authorize this chat in the TUI to continue.\n\nCommands:\n/id - Show Chat ID\n/status - Show daemon status\n/help - Show this help")
						msg.ParseMode = "Markdown"
						bot.Send(msg)
						if !isAllowed {
							warn := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ This chat (%d) is not authorized. Your messages will be ignored.", chatID))
							bot.Send(warn)
						}
						continue
					case "/status":
						if !isAllowed {
							continue
						}
						// We need a way to get status. onMessage is usually for tasks.
						// For now, we can use onMessage with a special prefix or just handle it here if we had access to butler.
						// But Start only gets onMessage.
						reply := onMessage("telegram", chatIDStr, from, "get_status_internal") // Hacky way if butler handles it
						msg := tgbotapi.NewMessage(chatID, "📊 *System Status*\n"+reply)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
						continue
					case "/help":
						msg := tgbotapi.NewMessage(chatID, "🦀 *Auracrab Help*\n\n- Send any text to start a new task.\n- Use `@crabname task` to delegate to a specific agent.\n- `/status`: Check daemon health.\n- `/id`: Get this chat's ID.")
						msg.ParseMode = "Markdown"
						bot.Send(msg)
						continue
					}
				}

				if !isAllowed {
					log.Printf("Ignored message from unauthorized chat: %d", chatID)
					msg := tgbotapi.NewMessage(chatID, "🚫 *Access Denied*\n\nThis chat is not authorized to interact with this Auracrab instance. Please contact the owner to authorize this Chat ID.")
					msg.ParseMode = "Markdown"
					_, _ = bot.Send(msg)
					continue
				}

				// Dispatch to Butler
				reply := onMessage("telegram", chatIDStr, from, text)
				if reply == "" {
					continue
				}
				if len(reply) > 4000 {
					reply = reply[:3997] + "..."
				}

				// Send reply
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
				msg.ReplyToMessageID = update.Message.MessageID

				if _, err := bot.Send(msg); err != nil {
					log.Printf("Error sending telegram message: %v", err)
				}
			}
		}
	}()

	return nil
}

func (t *TelegramChannel) Stop() error {
	t.saveOffset()
	return nil
}

func (t *TelegramChannel) Send(to string, text string) error {
	if t.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	var chatID int64
	_, err := fmt.Sscanf(to, "%d", &chatID)
	if err != nil {
		// If 'to' is not an ID, maybe it's a username?
		// tgbotapi doesn't easily support sending by username without a ChatID.
		return fmt.Errorf("invalid chat ID: %s", to)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	_, err = t.bot.Send(msg)
	return err
}

func (t *TelegramChannel) Broadcast(message string) error {
	if t.bot == nil || t.history == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	chats, err := t.history.ListAuthorizedEntities("telegram")
	if err != nil {
		return err
	}

	for _, chatIDStr := range chats {
		var chatID int64
		_, err := fmt.Sscanf(chatIDStr, "%d", &chatID)
		if err != nil {
			continue
		}
		msg := tgbotapi.NewMessage(chatID, message)
		_, _ = t.bot.Send(msg)
	}
	return nil
}

func (t *TelegramChannel) loadOffset() {
	path := filepath.Join(t.stateDir, "offset.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var state struct {
		LastUpdateID int `json:"last_update_id"`
	}
	if err := json.Unmarshal(data, &state); err == nil {
		t.offset = state.LastUpdateID
	}
}

func (t *TelegramChannel) saveOffset() {
	path := filepath.Join(t.stateDir, "offset.json")
	state := struct {
		LastUpdateID int `json:"last_update_id"`
	}{LastUpdateID: t.offset}
	data, _ := json.Marshal(state)
	_ = os.WriteFile(path, data, 0600)
}
