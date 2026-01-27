package connect

import (
	"os"

	"github.com/nathfavour/auracrab/pkg/vault"
)

func init() {
	v := vault.GetVault()

	// Automatically register Telegram if TOKEN is in environment or vault
	token, err := v.Get("TELEGRAM_TOKEN")
	if err != nil {
		token = os.Getenv("TELEGRAM_TOKEN")
	}
	if token != "" {
		RegisterChannel(&TelegramChannel{Token: token})
	}

	// Automatically register Discord if TOKEN is in environment or vault
	discordToken, err := v.Get("DISCORD_TOKEN")
	if err != nil {
		discordToken = os.Getenv("DISCORD_TOKEN")
	}
	if discordToken != "" {
		RegisterChannel(&DiscordChannel{Token: discordToken})
	}
}
