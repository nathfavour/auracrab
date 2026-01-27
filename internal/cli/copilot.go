package cli
package cli

import (


























}	rootCmd.AddCommand(copilotCmd)func init() {}	},		<-cmd.Context().Done()		// Wait for context cancellation or keep alive				}			os.Exit(1)			fmt.Printf("Failed to start agent: %v\n", err)		if err := agent.Start(cmd.Context()); err != nil {		fmt.Println("Starting Copilot SDK agent...")		agent := copilot.NewAgent()	Run: func(cmd *cobra.Command, args []string) {	Short: "Start Auracrab as a Copilot SDK agent",	Use:   "copilot",var copilotCmd = &cobra.Command{)	"github.com/spf13/cobra"	"github.com/nathfavour/auracrab/internal/copilot"	"os"	"fmt"