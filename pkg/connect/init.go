package connect

import (
	"os"
)

func init() {
	// Automatically register Telegram if TOKEN is in environment
	token := os.Getenv("TELEGRAM_TOKEN")
	if token != "" {
		RegisterChannel(&TelegramChannel{Token: token})
	}

	// Automatically register Discord if TOKEN is in environment
	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken != "" {
		RegisterChannel(&DiscordChannel{Token: discordToken})
	}
}
