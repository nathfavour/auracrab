package cli

import (
	"fmt"
	"os"

	"github.com/nathfavour/auracrab/internal/copilot"
	"github.com/spf13/cobra"
)

var copilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "Start Auracrab as a Copilot SDK agent",
	Run: func(cmd *cobra.Command, args []string) {
		agent := copilot.NewAgent()
		fmt.Println("Starting Copilot SDK agent...")
		if err := agent.Start(cmd.Context()); err != nil {
			fmt.Printf("Failed to start agent: %v\n", err)
			os.Exit(1)
		}

		// Wait for context cancellation or keep alive
		<-cmd.Context().Done()
	},
}

func init() {
	rootCmd.AddCommand(copilotCmd)
}
