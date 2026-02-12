package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nathfavour/auracrab/pkg/core"
	"github.com/nathfavour/auracrab/pkg/crabs"
	"github.com/nathfavour/auracrab/pkg/skills"
	"github.com/nathfavour/auracrab/pkg/vault"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(vibeManifestCmd)
	rootCmd.AddCommand(executeCmd)
	rootCmd.AddCommand(egoCmd)
}

var egoCmd = &cobra.Command{
	Use:   "ego",
	Short: "Check the status of Auracrab's Ego Module",
	Run: func(cmd *cobra.Command, args []string) {
		butler := core.GetButler()
		e := butler.Ego
		identity := e.GetIdentity()

		fmt.Printf("--- ENTITY IDENTITY ---\n")
		fmt.Printf("Name: %s\n", identity.Name)
		fmt.Printf("Born: %s\n", identity.BornAt.Format(time.RFC822))
		fmt.Printf("Vibe: %s\n", identity.Vibe)
		fmt.Printf("\n--- CORE DIRECTIVES ---\n")
		for _, d := range identity.CoreDirectives {
			fmt.Printf("- %s\n", d)
		}

		fmt.Printf("\n--- DRIVES ---\n")
		for _, d := range e.Drives {
			bar := strings.Repeat("█", int(d.Value*20)) + strings.Repeat("░", 20-int(d.Value*20))
			fmt.Printf("%-18s [%s] %.2f - %s\n", d.Name, bar, d.Value, d.Description)
		}

		fmt.Printf("\n--- RECENT THOUGHTS ---\n")
		start := len(e.Narrative) - 10
		if start < 0 {
			start = 0
		}
		for _, t := range e.Narrative[start:] {
			fmt.Println(t)
		}
	},
}

var vibeManifestCmd = &cobra.Command{
	Use:   "vibe-manifest",
	Short: "Output vibe manifest for vibeauracle",
	Run: func(cmd *cobra.Command, args []string) {
		toolSet := []map[string]interface{}{
			{
				"name":        "sys_tool_wand",
				"description": "Discover and search for all available system tools. Use this if you are unsure which tool to use or need to explore capabilities.",
				"inputSchema": json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {"type": "string", "description": "Search query for tools (e.g. 'file', 'git', 'network')"}
					},
					"required": ["query"]
				}`),
			},
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
				"description": "Check vibeauracle health via UDS and report issues",
				"inputSchema": json.RawMessage(`{"type":"object","properties":{}}`),
			},
			{
				"name":        "auracrab_register_crab",
				"description": "Register a new user-defined Crab agent",
				"inputSchema": json.RawMessage(`{
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"name": {"type": "string"},
						"description": {"type": "string"},
						"instructions": {"type": "string"},
						"skills": {"type": "array", "items": {"type": "string"}}
					},
					"required": ["id", "name", "instructions"]
				}`),
			},
		}

		// Add registered Crabs as specialized tools
		reg, _ := crabs.NewRegistry()
		crabList, _ := reg.List()
		for _, c := range crabList {
			toolSet = append(toolSet, map[string]interface{}{
				"name":        "auracrab_delegate_" + c.ID,
				"description": fmt.Sprintf("Delegate a task to specialized agent '%s': %s", c.Name, c.Description),
				"inputSchema": json.RawMessage(`{"type":"object","properties":{"task":{"type":"string","description":"Task for the agent"}}}`),
			})
		}

		// Add dynamic skills
		v := vault.GetVault()
		for _, s := range skills.GetRegistry().List() {
			enabled, _ := v.Get(strings.ToUpper(s.Name()) + "_ENABLED")
			if enabled != "" && enabled != "true" {
				continue
			}

			var manifestMap map[string]interface{}
			_ = json.Unmarshal(s.Manifest(), &manifestMap)

			// Extract properties from the manifest's parameters
			inputSchema := manifestMap["parameters"]
			if inputSchema == nil {
				inputSchema = json.RawMessage(`{"type":"object","properties":{}}`)
			}

			toolSet = append(toolSet, map[string]interface{}{
				"name":        s.Name(),
				"description": s.Description(),
				"inputSchema": inputSchema,
			})
		}

		manifest := map[string]interface{}{
			"id":          "auracrab",
			"name":        "Auracrab",
			"repo":        "nathfavour/auracrab",
			"description": "Autonomous AI Agent Daemon. INSTRUCTIONS: You have access to a wide array of system tools via the 'sys_tool_wand'. If a task requires file system access, shell execution, or web browsing, use the wand to find the appropriate tool first. Always aim for completion.",
			"protocol":    "stdio",
			"command":     "auracrab",
			"update_cmd":  "auracrab update",
			"inbuilt":     true,
			"comms": map[string]bool{
				"tui": true,
				"cli": true,
				"uds": true,
			},
			"capabilities": map[string]bool{
				"agentic":   true,
				"read_only": false,
			},
			"tool_set": toolSet,
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

		// Try dynamic skills first
		if s, ok := skills.GetRegistry().Get(toolName); ok {
			v := vault.GetVault()
			enabled, _ := v.Get(strings.ToUpper(s.Name()) + "_ENABLED")
			if enabled != "" && enabled != "true" {
				fmt.Printf(`{"content": "Error: skill '%s' is disabled", "status": "error"}`+"\n", toolName)
				return
			}

			var argData json.RawMessage
			if len(args) > 1 {
				argData = json.RawMessage(args[1])
			}
			res, err := s.Execute(context.Background(), argData)
			if err != nil {
				fmt.Printf(`{"content": "Error: %v", "status": "error"}`+"\n", err)
				return
			}
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", res)
			return
		}

		switch {
		case toolName == "auracrab_status":
			status := butler.GetStatus()
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", status)
		case toolName == "auracrab_start_task":
			var params struct {
				Task string `json:"task"`
			}
			if len(args) > 1 {
				_ = json.Unmarshal([]byte(args[1]), &params)
			}

			task, err := butler.StartTask(cmd.Context(), params.Task, "")
			if err != nil {
				fmt.Printf(`{"content": "Error: %v", "status": "error"}`+"\n", err)
				return
			}
			fmt.Printf(`{"content": "Task started: %s (ID: %s)", "status": "success"}`+"\n", params.Task, task.ID)
		case toolName == "auracrab_list_tasks":
			tasks := butler.ListTasks()
			data, _ := json.Marshal(tasks)
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", string(data))
		case toolName == "auracrab_watch_health":
			health := butler.WatchHealth()
			fmt.Printf(`{"content": %q, "status": "success"}`+"\n", health)
		case toolName == "auracrab_register_crab":
			var crab crabs.Crab
			if len(args) > 1 {
				if err := json.Unmarshal([]byte(args[1]), &crab); err != nil {
					fmt.Printf(`{"content": "Error parsing crab: %v", "status": "error"}`+"\n", err)
					return
				}
			}
			reg, err := crabs.NewRegistry()
			if err != nil {
				fmt.Printf(`{"content": "Error accessing registry: %v", "status": "error"}`+"\n", err)
				return
			}
			if err := reg.Register(crab); err != nil {
				fmt.Printf(`{"content": "Error registering crab: %v", "status": "error"}`+"\n", err)
				return
			}
			fmt.Printf(`{"content": "Crab '%s' registered successfully.", "status": "success"}`+"\n", crab.Name)
		case strings.HasPrefix(toolName, "auracrab_delegate_"):
			crabID := strings.TrimPrefix(toolName, "auracrab_delegate_")
			var params struct {
				Task string `json:"task"`
			}
			if len(args) > 1 {
				_ = json.Unmarshal([]byte(args[1]), &params)
			}

			reg, _ := crabs.NewRegistry()
			crab, err := reg.Get(crabID)
			if err != nil {
				fmt.Printf(`{"content": "Error finding crab: %v", "status": "error"}`+"\n", err)
				return
			}

			// For delegation, we actually start a nested task or just return instructions.
			// Ideally, we want vibeauracle to continue with the crab's instructions.
			fmt.Printf(`{"content": "Agent %s instructions: %s. Context: %s", "status": "success"}`+"\n", crab.Name, crab.Instructions, params.Task)

		default:
			fmt.Printf("Unknown tool: %s\n", toolName)
			os.Exit(1)
		}
	},
}
