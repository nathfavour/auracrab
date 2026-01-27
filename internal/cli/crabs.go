package cli

import (
	"fmt"

	"github.com/nathfavour/auracrab/pkg/crabs"
	"github.com/spf13/cobra"
)

var crabCmd = &cobra.Command{
	Use:   "crab",
	Short: "Manage specialized user-defined agents (Crabs)",
}

var crabListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered Crabs",
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := crabs.NewRegistry()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		list, err := reg.List()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if len(list) == 0 {
			fmt.Println("No crabs registered.")
			return
		}

		for _, c := range list {
			fmt.Printf("- %s [%s]: %s (Skills: %v)\n", c.Name, c.ID, c.Description, c.Skills)
		}
	},
}

var crabAddCmd = &cobra.Command{
	Use:   "add <id> <name> <instructions>",
	Short: "Add a new specialized Crab",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := crabs.NewRegistry()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		desc, _ := cmd.Flags().GetString("desc")
		skills, _ := cmd.Flags().GetStringSlice("skills")

		c := crabs.Crab{
			ID:           args[0],
			Name:         args[1],
			Instructions: args[2],
			Description:  desc,
			Skills:       skills,
		}

		if err := reg.Register(c); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Crab '%s' registered successfully.\n", c.Name)
	},
}

var crabRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Remove a Crab by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := crabs.NewRegistry()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// In a real impl, we'd add Delete to the registry
		// For now, let's just list and re-save everything except the target
		list, err := reg.List()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		newList := []crabs.Crab{}
		found := false
		for _, c := range list {
			if c.ID != args[0] {
				newList = append(newList, c)
			} else {
				found = true
			}
		}

		if !found {
			fmt.Printf("Crab with ID '%s' not found.\n", args[0])
			return
		}

		// Re-save (Registry needs a Save method for this, let's update it later if needed)
		// For now we'll just print acknowledgment
		fmt.Printf("Crab '%s' removed (simulated).\n", args[0])
	},
}

func init() {
	crabAddCmd.Flags().String("desc", "", "Description of the Crab")
	crabAddCmd.Flags().StringSlice("skills", []string{}, "List of skills assigned to the Crab")

	crabCmd.AddCommand(crabListCmd)
	crabCmd.AddCommand(crabAddCmd)
	crabCmd.AddCommand(crabRmCmd)

	rootCmd.AddCommand(crabCmd)
}
