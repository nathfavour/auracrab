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
	enabled, _ := v.Get("TELEGRAM_ENABLED")
	if token != "" && (enabled == "" || enabled == "true") {
		RegisterChannel(&TelegramChannel{Token: token})
	}

	// Automatically register Discord if TOKEN is in environment or vault
	discordToken, err := v.Get("DISCORD_TOKEN")
	if err != nil {
		discordToken = os.Getenv("DISCORD_TOKEN")
	}
	discordEnabled, _ := v.Get("DISCORD_ENABLED")
	if discordToken != "" && (discordEnabled == "" || discordEnabled == "true") {
		RegisterChannel(&DiscordChannel{Token: discordToken})
	}

	// Register Browser Channel
	RegisterChannel(NewBrowserChannel(9999))
}
