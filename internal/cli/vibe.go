package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(vibeManifestCmd)
	rootCmd.AddCommand(executeCmd)
}

var vibeManifestCmd = &cobra.Command{
	Use:   "vibe-manifest",
	Short: "Output vibe manifest for vibeauracle",
	Run: func(cmd *cobra.Command, args []string) {
		manifest := map[string]interface{}{
			"id":          "auracrab",
			"name":        "Auracrab",
			"repo":        "nathfavour/auracrab",
			"version":     Version,
			"description": "Autonomous, persistent AI agent daemon",
			"protocol":    "stdio",
			"command":     "auracrab",
			"update_cmd":  "auracrab update",
			"inbuilt":     true,
			"tool_set": []map[string]interface{}{
				{
					"name":        "auracrab_status",
					"description": "Get the current status of the Auracrab daemon",
					"inputSchema": json.RawMessage(`{"type":"object","properties":{}}`),
				},
				{
					"name":        "auracrab_start_task",
					"description": "Start a new autonomous task",
					"inputSchema": json.RawMessage(`{"type":"object","properties":{"task":{"type":"string","description":"Task description"}}}`),
				},
			},
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		fmt.Println(string(data))
	},
}

var executeCmd = &cobra.Command{
	Use:   "execute [tool] [args]",
	Short: "Execute a tool in vibe mode",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Tool name required")
			os.Exit(1)
		}
		
		toolName := args[0]
		
		switch toolName {
		case "auracrab_status":
			fmt.Println(`{"content": "Auracrab is idling.", "status": "success"}`)
		case "auracrab_start_task":
			fmt.Println(`{"content": "Task started successfully.", "status": "success"}`)
		default:
			fmt.Printf("Unknown tool: %s\n", toolName)
			os.Exit(1)
		}
	},
}
