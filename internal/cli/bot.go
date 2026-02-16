package cli

import (
	"fmt"

	"github.com/nathfavour/auracrab/pkg/social"
	"github.com/spf13/cobra"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Manage messaging bots (Telegram/Discord)",
}

var botAddCmd = &cobra.Command{
	Use:   "add [name] [token]",
	Short: "Register a new bot",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		platform, _ := cmd.Flags().GetString("platform")
		bm := social.GetBotManager()

		cfg := social.BotConfig{
			Name:     args[0],
			Token:    args[1],
			Platform: platform,
		}

		if err := bm.AddBot(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Bot %s [%s] added successfully.\n", args[0], platform)
	},
}

var botListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered bots",
	Run: func(cmd *cobra.Command, args []string) {
		bm := social.GetBotManager()
		bots := bm.ListBots()

		if len(bots) == 0 {
			fmt.Println("No bots registered.")
			return
		}

		for _, b := range bots {
			owner := "Unregistered"
			if b.OwnerID != "" {
				owner = b.OwnerID
			}
			fmt.Printf("- %s [%s] (Owner: %s, Mode: %s)\n", b.Name, b.Platform, owner, b.Mode)
		}
	},
}

func init() {
	botAddCmd.Flags().StringP("platform", "p", "telegram", "Platform for the bot (telegram/discord)")

	botCmd.AddCommand(botAddCmd)
	botCmd.AddCommand(botListCmd)
	rootCmd.AddCommand(botCmd)
}
