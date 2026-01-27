package connect

import (
	"context"
	"log"
)

// TelegramChannel is a placeholder for a real Telegram integration.
type TelegramChannel struct {
	Token string
}

func (t *TelegramChannel) Name() string {
	return "telegram"
}

func (t *TelegramChannel) Start(ctx context.Context, onMessage func(from string, text string) string) error {
	log.Printf("Connecting to Telegram (Token: %s...)", t.Token[:5])
	// In a real implementation, we would use a library like telebot or similar.
	
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (t *TelegramChannel) Stop() error {
	return nil
}

func init() {
	// Telegram integration will be registered when the token is provided via config
}
