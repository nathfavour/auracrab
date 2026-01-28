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
	"github.com/nathfavour/auracrab/pkg/vault"
)

// TelegramChannel is a real Telegram integration using long-polling.
type TelegramChannel struct {
	Token    string
	offset   int
	stateDir string
}

func (t *TelegramChannel) Name() string {
	return "telegram"
}

func (t *TelegramChannel) Start(ctx context.Context, onMessage func(from string, text string) string) error {
	home, _ := os.UserHomeDir()
	t.stateDir = filepath.Join(home, ".local", "share", "auracrab", "telegram")
	_ = os.MkdirAll(t.stateDir, 0755)

	t.loadOffset()

	bot, err := tgbotapi.NewBotAPI(t.Token)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %v", err)
	}

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
				v := vault.GetVault()
				allowedChats, _ := v.Get("TELEGRAM_ALLOWED_CHATS")
				
				isAllowed := false
				if allowedChats == "" {
					// If none configured, allow but log a warning (or we could default to private only?)
					// Let's default to allowing but providing a way to lock it down.
					isAllowed = true
				} else {
					for _, idStr := range strings.Split(allowedChats, ",") {
						if strings.TrimSpace(idStr) == fmt.Sprintf("%d", chatID) {
							isAllowed = true
							break
						}
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
						msg := tgbotapi.NewMessage(chatID, "🦀 *Auracrab Telegram Bot*\n\nI am your autonomous agent. Send me tasks or delegate to specific crabs using @crabname.\n\nCommands:\n/id - Show Chat ID\n/status - Show daemon status\n/help - Show this help")
						msg.ParseMode = "Markdown"
						bot.Send(msg)
						if !isAllowed {
							warn := tgbotapi.NewMessage(chatID, "⚠️ This chat is not in the allowed list. Your messages will be ignored until this Chat ID is added to `TELEGRAM_ALLOWED_CHATS`.")
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
						reply := onMessage(from, "get_status_internal") // Hacky way if butler handles it
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
					continue
				}

				// Dispatch to Butler
				reply := onMessage(from, text)
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
