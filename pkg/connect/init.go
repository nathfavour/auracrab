package connect

import (
	"os"
)

func init() {
	// Automatically register Telegram if TOKEN is in environment
	// In a real app, this would come from a proper config file ~/.config/auracrab/config.yaml
	token := os.Getenv("TELEGRAM_TOKEN")
	if token != "" {
		RegisterChannel(&TelegramChannel{Token: token})
	}
}
