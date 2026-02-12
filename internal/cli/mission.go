package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/nathfavour/auracrab/pkg/core"
	"github.com/spf13/cobra"
)

var missionCmd = &cobra.Command{
	Use:   "mission",
	Short: "Manage autonomous missions and deadlines",
}

var missionCreateCmd = &cobra.Command{
	Use:   "create [title] [goal] [deadline]",
	Short: "Create a new mission",
	Long:  "Create a mission. Deadline format: 2006-01-02T15:04:05Z07:00",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		deadline, err := time.Parse(time.RFC3339, args[2])
		if err != nil {
			fmt.Printf("Invalid deadline format: %v\n", err)
			os.Exit(1)
		}

		b := core.GetButler()
		m := b.Missions.CreateMission(args[0], "", args[1], deadline)
		fmt.Printf("Mission created: %s (%s)\n", m.Title, m.ID)
	},
}

var missionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current active mission status",
	Run: func(cmd *cobra.Command, args []string) {
		b := core.GetButler()
		m := b.Missions.GetActiveMission()
		if m == nil {
			fmt.Println("No active mission.")
			return
		}

		tr, _ := b.Missions.TimeRemaining(m.ID)
		fmt.Printf("🚀 MISSION: %s\n", m.Title)
		fmt.Printf("🎯 GOAL:    %s\n", m.Goal)
		fmt.Printf("⏳ REMAINING: %v\n", tr.Round(time.Second))
		fmt.Printf("📊 PROGRESS:  %.1f%%\n", m.Progress*100)
		fmt.Printf("🧠 EST. TTC:  %v\n", m.EstimatedTTC)
	},
}

func init() {
	missionCmd.AddCommand(missionCreateCmd)
	missionCmd.AddCommand(missionStatusCmd)
	rootCmd.AddCommand(missionCmd)
}
