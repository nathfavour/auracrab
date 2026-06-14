package cli

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(StopCmd)
}

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running Auracrab daemon",
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := config.PIDPath()
		if pidData, err := os.ReadFile(pidFile); err == nil {
			pid, _ := strconv.Atoi(string(pidData))
			if isProcessRunning(pid) {
				process, _ := os.FindProcess(pid)
				if err := process.Signal(syscall.SIGTERM); err != nil {
					fmt.Printf("Error stopping daemon process %d: %v\n", pid, err)
				} else {
					fmt.Printf("🦀 Auracrab daemon (PID: %d) stopped successfully.\n", pid)
				}
			} else {
				fmt.Println("🦀 Auracrab is not running.")
			}
			_ = os.Remove(pidFile)
		} else {
			fmt.Println("🦀 No running Auracrab daemon detected (PID file missing).")
		}
	},
}
