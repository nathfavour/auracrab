package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nathfavour/auracrab/pkg/social"
	"github.com/nathfavour/auracrab/pkg/vault"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(SocialCmd)
	SocialCmd.AddCommand(SocialStatusCmd)
	SocialCmd.AddCommand(SocialEnableCmd)
	SocialCmd.AddCommand(SocialDisableCmd)
	SocialCmd.AddCommand(SocialSetPlatformsCmd)
	SocialCmd.AddCommand(SocialSetIntervalCmd)
	SocialCmd.AddCommand(SocialSetPromptCmd)
	SocialCmd.AddCommand(SocialInteractiveConfigCmd)
}

var SocialCmd = &cobra.Command{
	Use:   "social",
	Short: "Configure continuous social posting daemon",
	Long:  `Manage and configure continuous social updates generation and cross-posting to Threads and other platforms.`,
}

var SocialStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current social configuration status",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := social.LoadSocialConfig()
		if err != nil {
			fmt.Printf("Error loading social config: %v\n", err)
			return
		}
		fmt.Printf("🦀 Continuous Social Poster Status:\n")
		fmt.Printf("  Enabled:        %t\n", cfg.Enabled)
		fmt.Printf("  Platforms:      %s\n", strings.Join(cfg.Platforms, ", "))
		fmt.Printf("  Post Interval:  %v\n", cfg.PostInterval)
		fmt.Printf("  Prompt:         %q\n", cfg.Prompt)
	},
}

var SocialEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable the social poster daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := social.LoadSocialConfig()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		cfg.Enabled = true
		if err := social.SaveSocialConfig(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("🦀 Continuous social poster enabled.")
	},
}

var SocialDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable the social poster daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := social.LoadSocialConfig()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		cfg.Enabled = false
		if err := social.SaveSocialConfig(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("🦀 Continuous social poster disabled.")
	},
}

var SocialSetPlatformsCmd = &cobra.Command{
	Use:   "set-platforms [platforms]",
	Short: "Set platforms (comma separated, e.g., threads,x)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := social.LoadSocialConfig()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		platforms := strings.Split(args[0], ",")
		for i, p := range platforms {
			platforms[i] = strings.TrimSpace(p)
		}
		cfg.Platforms = platforms
		if err := social.SaveSocialConfig(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("🦀 Social platforms updated to: %v\n", cfg.Platforms)
	},
}

var SocialSetIntervalCmd = &cobra.Command{
	Use:   "set-interval [duration]",
	Short: "Set posting interval (e.g., 30s, 1h, 6h)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := social.LoadSocialConfig()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		d, err := time.ParseDuration(args[0])
		if err != nil {
			fmt.Printf("Invalid duration %q: %v\n", args[0], err)
			return
		}
		if d < 10*time.Second {
			fmt.Println("Error: posting interval must be at least 10 seconds.")
			return
		}
		cfg.PostInterval = d
		if err := social.SaveSocialConfig(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("🦀 Social posting interval updated to: %v\n", cfg.PostInterval)
	},
}

var SocialSetPromptCmd = &cobra.Command{
	Use:   "set-prompt [prompt]",
	Short: "Set AI post generation prompt",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := social.LoadSocialConfig()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		cfg.Prompt = args[0]
		if err := social.SaveSocialConfig(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("🦀 Social posting prompt updated.")
	},
}

var SocialInteractiveConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure social media platform keys interactively",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		v := vault.GetVault()

		fmt.Println("🦀 Supported Social Platforms:")
		fmt.Println("  1) Meta Threads")
		fmt.Println("  2) X (formerly Twitter)")
		fmt.Println("  3) LinkedIn")
		fmt.Println("  4) Facebook")
		fmt.Println("  5) Instagram")
		fmt.Println("  6) Cancel")
		fmt.Print("Select a platform to configure (1-6): ")

		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)

		var platformName string
		switch choiceStr {
		case "1", "threads", "Threads":
			platformName = "threads"
		case "2", "x", "X":
			platformName = "x"
		case "3", "linkedin", "LinkedIn":
			platformName = "linkedin"
		case "4", "facebook", "Facebook":
			platformName = "facebook"
		case "5", "instagram", "Instagram":
			platformName = "instagram"
		default:
			fmt.Println("Configuration cancelled.")
			return
		}

		fmt.Printf("\n--- Configuring %s ---\n", strings.Title(platformName))

		switch platformName {
		case "threads":
			fmt.Print("Enter Threads Access Token (leave empty to skip/keep existing): ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token != "" {
				_ = v.Set("THREADS_ACCESS_TOKEN", token)
			}

			fmt.Print("Enter Threads User ID (leave empty to skip/keep existing, or 'me'): ")
			uid, _ := reader.ReadString('\n')
			uid = strings.TrimSpace(uid)
			if uid != "" {
				_ = v.Set("THREADS_USER_ID", uid)
			}

		case "x":
			fmt.Print("Enter X API Key / Access Token: ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token != "" {
				_ = v.Set("X_API_KEY", token)
			}

		case "linkedin":
			fmt.Print("Enter LinkedIn Access Token: ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token != "" {
				_ = v.Set("LINKEDIN_ACCESS_TOKEN", token)
			}

		case "facebook":
			fmt.Print("Enter Facebook Access Token: ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token != "" {
				_ = v.Set("FACEBOOK_ACCESS_TOKEN", token)
			}

		case "instagram":
			fmt.Print("Enter Instagram Access Token: ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token != "" {
				_ = v.Set("INSTAGRAM_ACCESS_TOKEN", token)
			}
		}

		// Offer to enable this platform in the config
		fmt.Printf("Would you like to add %s to the active posting platforms list? (y/n): ", platformName)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.ToLower(strings.TrimSpace(confirm))
		if confirm == "y" || confirm == "yes" {
			cfg, err := social.LoadSocialConfig()
			if err != nil {
				fmt.Printf("Error updating config: %v\n", err)
				return
			}
			exists := false
			for _, p := range cfg.Platforms {
				if p == platformName {
					exists = true
					break
				}
			}
			if !exists {
				cfg.Platforms = append(cfg.Platforms, platformName)
				if err := social.SaveSocialConfig(cfg); err != nil {
					fmt.Printf("Error saving config: %v\n", err)
					return
				}
			}
			fmt.Printf("🦀 %s successfully added to your active platforms list.\n", strings.Title(platformName))
		}

		fmt.Println("🦀 Configuration complete.")
	},
}
