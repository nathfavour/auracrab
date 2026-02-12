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
		// If managed by Anyisland, we skip internal update checks as Anyisland handles this via Pulse/OTA
		if anyisland.IsManaged() {
			return
		}

		// Clean up update signals if we are now on the new version
		availFile := filepath.Join(config.DataDir(), ".update_available")
		completeFile := filepath.Join(config.DataDir(), ".update_complete")
		if data, err := os.ReadFile(availFile); err == nil {
			remoteSHA := strings.TrimSpace(string(data))
			if strings.HasPrefix(remoteSHA, config.Commit) && config.Commit != "none" {
				_ = os.Remove(availFile)
				_ = os.Remove(completeFile)
			}
		}

		// Start background update check
		go func() {
			if strings.Contains(os.Args[0], "go-build") {
				return // Don't check during development builds
			}
			if cmd.Name() == "update" || cmd.Name() == "version" {
				return // Don't check if we're already updating or just checking version
			}
			
			checkFile := filepath.Join(config.DataDir(), ".last_update_check")
			if stat, err := os.Stat(checkFile); err == nil {
				if time.Since(stat.ModTime()) < 1*time.Hour {
					return // Checked recently
				}
			}
			
			// Touch check file
			_ = os.WriteFile(checkFile, []byte(time.Now().String()), 0644)

			// Lightweight remote check
			repoURL := "https://github.com/nathfavour/auracrab.git"
			remoteCmd := exec.Command("git", "ls-remote", repoURL, "HEAD")
			out, err := remoteCmd.Output()
			if err != nil {
				return
			}
			
			remoteSHA := strings.Fields(string(out))[0]
			if len(remoteSHA) > 7 && config.Commit != "none" && !strings.HasPrefix(remoteSHA, config.Commit) {
				// Update available!
				availFile := filepath.Join(config.DataDir(), ".update_available")
				_ = os.WriteFile(availFile, []byte(remoteSHA), 0644)

				// Run installation in background
				// We use the same install.sh which is already smart
				go func() {
					scriptPath := filepath.Join(config.SourceDir(), "install.sh")
					completeFile := filepath.Join(config.DataDir(), ".update_complete")
					
					var err error
					if _, err = os.Stat(scriptPath); err != nil {
						// Fallback to curl
						err = exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/nathfavour/auracrab/master/install.sh | bash").Run()
					} else {
						err = exec.Command("bash", scriptPath).Run()
					}
					
					if err == nil {
						_ = os.WriteFile(completeFile, []byte(time.Now().String()), 0644)
					}
				}()
			}
		}()
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
