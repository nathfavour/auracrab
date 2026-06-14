package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/nathfavour/auracrab/pkg/social"
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
