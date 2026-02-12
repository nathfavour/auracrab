package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nathfavour/auracrab/pkg/anyisland"
	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "auracrab",
	Short: "auracrab is a modular CLI tool",
	Long:  `A highly modular CLI project structure built with Go and Cobra.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", config.Version, config.Commit, config.BuildDate),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// managed_updates are handled by Butler in autonomous mode via Anyisland Pulse
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Zero-command entry point: Start the autonomous heartbeat
		fmt.Println("🦀 Auracrab is coming alive...")
		
		// Run the butler in continuous mode
		ctx := context.Background()
		butler := core.GetButler()
		
		if err := butler.Serve(ctx); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.auracrab.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".auracrab")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
