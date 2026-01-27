package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nathfavour/auracrab/pkg/core"
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
				{
					"name":        "auracrab_list_tasks",
					"description": "List all tasks managed by Auracrab",
					"inputSchema": json.RawMessage(`{"type":"object","properties":{}}`),
				},
				{
					"name":        "auracrab_watch_health",
					"description": "Watch vibeauracle health logs and report issues",
					"inputSchema": json.RawMessage(`{"type":"object","properties":{}}`),
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
		butler := core.GetButler()
		
		switch toolName {
		case "auracrab_status":
			status := butler.GetStatus()
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", status)
		case "auracrab_start_task":
			var params struct {
				Task string `json:"task"`
			}
			if len(args) > 1 {
				_ = json.Unmarshal([]byte(args[1]), &params)
			}
			
			task, err := butler.StartTask(cmd.Context(), params.Task)
			if err != nil {
				fmt.Printf(`{"content": "Error: %v", "status": "error"}`+"\n", err)
				return
			}
			fmt.Printf(`{"content": "Task started: %s (ID: %s)", "status": "success"}`+"\n", params.Task, task.ID)
		case "auracrab_list_tasks":
			tasks := butler.ListTasks()
			data, _ := json.Marshal(tasks)
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", string(data))
		case "auracrab_watch_health":
			health := butler.WatchHealth()
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", health)
		default:
			fmt.Printf("Unknown tool: %s\n", toolName)
			os.Exit(1)
		}
	},
}
