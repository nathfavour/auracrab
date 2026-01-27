package connect

import (
"context"
"encoding/json"
"fmt"
"log"
"os"
"path/filepath"

tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

				log.Printf("[%s] %s", from, text)

				// Dispatch to Butler
				reply := onMessage(from, text)

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
