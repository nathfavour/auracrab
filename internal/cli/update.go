package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update auracrab and its dependencies to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Current Version: %s\n", Version)
		fmt.Println("Checking for updates...")

		// Use local install.sh if we are in source directory for faster/smarter updates
		scriptPath := "./install.sh"
		if _, err := os.Stat(scriptPath); err != nil {
			// Fallback to remote
			scriptPath = "https://raw.githubusercontent.com/nathfavour/auracrab/master/install.sh"
			fmt.Println("Running remote installer...")
			updateCmd := exec.Command("bash", "-c", "curl -fsSL "+scriptPath+" | bash")
			updateCmd.Stdout = os.Stdout
			updateCmd.Stderr = os.Stderr
			if err := updateCmd.Run(); err != nil {
				fmt.Printf("\n❌ Update failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Running local installer...")
			updateCmd := exec.Command("bash", scriptPath)
			updateCmd.Stdout = os.Stdout
			updateCmd.Stderr = os.Stderr
			if err := updateCmd.Run(); err != nil {
				fmt.Printf("\n❌ Update failed: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Println("\n✨ Update process completed!")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
