package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const auracrabRepo = "nathfavour/auracrab"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update auracrab to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking for updates...")
		
		// 1. Ensure vibeauracle is installed/updated
		fmt.Println("Ensuring vibeauracle is up to date...")
		vibeUpdate := exec.Command("vibeaura", "update")
		vibeUpdate.Stdout = os.Stdout
		vibeUpdate.Stderr = os.Stderr
		if err := vibeUpdate.Run(); err != nil {
			fmt.Println("Note: vibeaura update failed or not found, attempting re-install...")
			installVibe := exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | bash")
			installVibe.Stdout = os.Stdout
			installVibe.Stderr = os.Stderr
			installVibe.Run()
		}

		// 2. Update Auracrab (Self-update)
		// For now, we'll use a simple approach similar to the install script
		fmt.Println("Updating auracrab...")
		osName := runtime.GOOS
		archName := runtime.GOARCH
		
		// Map arch to release names
		if archName == "amd64" {
			archName = "amd64"
		} else if archName == "arm64" {
			archName = "arm64"
		}

		binaryName := fmt.Sprintf("auracrab-%s-%s", osName, archName)
		downloadURL := fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", auracrabRepo, binaryName)

		fmt.Printf("Downloading %s...\n", binaryName)
		
		tmpPath := "/tmp/auracrab_update"
		downloadCmd := exec.Command("curl", "-L", downloadURL, "-o", tmpPath)
		if err := downloadCmd.Run(); err != nil {
			fmt.Printf("Error downloading update: %v\n", err)
			return
		}

		os.Chmod(tmpPath, 0755)

		exePath, err := os.Executable()
		if err != nil {
			fmt.Printf("Error finding executable: %v\n", err)
			return
		}

		fmt.Printf("Installing to %s...\n", exePath)
		
		// Attempt to move, use sudo if necessary
		mvCmd := exec.Command("mv", tmpPath, exePath)
		if err := mvCmd.Run(); err != nil {
			fmt.Println("Requesting sudo for installation...")
			sudoMv := exec.Command("sudo", "mv", tmpPath, exePath)
			sudoMv.Stdout = os.Stdout
			sudoMv.Stderr = os.Stderr
			if err := sudoMv.Run(); err != nil {
				fmt.Printf("Update failed: %v\n", err)
				return
			}
		}

		fmt.Println("Auracrab updated successfully!")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
