package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	isDaemon bool
	verbose  bool
	kill     bool
)

var rootCmd = &cobra.Command{
	Use:     "auracrab",
	Short:   "auracrab is a modular CLI tool",
	Long:    `A highly modular CLI project structure built with Go and Cobra.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", config.Version, config.Commit, config.BuildDate),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// managed_updates are handled by Butler in autonomous mode via Anyisland Pulse
	},
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := config.PIDPath()

		// 0. Handle Kill Flag
		if kill {
			if pidData, err := os.ReadFile(pidFile); err == nil {
				pid, _ := strconv.Atoi(string(pidData))
				if isProcessRunning(pid) {
					process, _ := os.FindProcess(pid)
					if err := process.Signal(syscall.SIGTERM); err != nil {
						fmt.Printf("Error killing process %d: %v\n", pid, err)
					} else {
						fmt.Printf("🦀 Auracrab (PID: %d) terminated.\n", pid)
					}
				} else {
					fmt.Println("🦀 Auracrab is not running.")
				}
				_ = os.Remove(pidFile)
			} else {
				fmt.Println("🦀 No PID file found. Auracrab is likely not running.")
			}
			return
		}

		// 1. Check if already running
		if pidData, err := os.ReadFile(pidFile); err == nil {
			pid, _ := strconv.Atoi(string(pidData))
			if isProcessRunning(pid) {
				fmt.Printf("🦀 Auracrab is already running (PID: %d)\n", pid)
				return
			}
		}

		// 2. Handle Daemonization
		if !isDaemon && !verbose {
			// Re-exec as daemon
			cmd := exec.Command(os.Args[0], "--daemon")
			// Redirect stdout/stderr to a log file if not verbose
			logFile, _ := os.OpenFile(filepath.Join(config.DataDir(), "auracrab.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			
			if err := cmd.Start(); err != nil {
				fmt.Printf("Error starting daemon: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("🦀 Auracrab started in background (PID: %d)\n", cmd.Process.Pid)
			return
		}

		// 3. Main process logic
		if verbose {
			fmt.Println("🦀 Auracrab is coming alive (verbose mode)...")
		}

		// Save PID
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
		defer os.Remove(pidFile)

		// Run the butler in continuous mode
		ctx := context.Background()
		butler := core.GetButler()

		if err := butler.Serve(ctx); err != nil {
			if verbose {
				fmt.Printf("Error: %v\n", err)
			}
			os.Exit(1)
		}
	},
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Need to send signal 0.
	err = process.Signal(syscall.Signal(0))
	return err == nil
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
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (run in foreground)")
	rootCmd.Flags().BoolVarP(&kill, "kill", "k", false, "kill the running auracrab process")
	rootCmd.Flags().BoolVar(&isDaemon, "daemon", false, "internal daemon flag")
	_ = rootCmd.Flags().MarkHidden("daemon")
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
