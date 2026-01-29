package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update auracrab and its dependencies to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Current Version: %s\n", config.Version)
		fmt.Println("Checking for updates...")

		updateScript := "curl -fsSL https://raw.githubusercontent.com/nathfavour/auracrab/master/install.sh | bash"

		updateCmd := exec.Command("bash", "-c", updateScript)
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr

		if err := updateCmd.Run(); err != nil {
			fmt.Printf("\n❌ Update failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✨ Update process completed!")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
