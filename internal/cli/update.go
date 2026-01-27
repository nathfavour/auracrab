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

		// Seamless update strategy: Run the universal installer.
		// This handles dependency management (vibeaura), platform detection,
		// and proper path installation in one reliable step.
		updateScript := "curl -fsSL https://raw.githubusercontent.com/nathfavour/auracrab/master/install.sh | bash"

		fmt.Println("Running universal installer...")
		updateCmd := exec.Command("bash", "-c", updateScript)
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr

		if err := updateCmd.Run(); err != nil {
			fmt.Printf("\n❌ Update failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✨ Auracrab and dependencies updated successfully!")
		fmt.Print("New Version: ")

		// Verify
		verifyCmd := exec.Command("auracrab", "version")
		verifyCmd.Stdout = os.Stdout
		verifyCmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
