package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/nathfavour/auracrab/pkg/ecosystem"
	"github.com/spf13/cobra"
)

var stackCategory string

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Install internal agentic libraries and well-known tools",
	Long: `Install polygeist and other stack components without cloning github.com/nathfavour/polygeist.

Internal libraries (polygeist, anyisland, vibeaura, auracrab) and well-known tools
(go, node, git, docker, ...) are installed through anyisland official packages.`,
}

var stackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List supported internal libraries and tools",
	Run: func(cmd *cobra.Command, args []string) {
		printLibraries(ecosystem.InternalLibraries, "Internal libraries")
		if stackCategory == "" || stackCategory == "tool" {
			printLibraries(ecosystem.WellKnownTools, "Well-known tools")
		}
	},
}

var stackInstallCmd = &cobra.Command{
	Use:   "install [name...]",
	Short: "Install one or more libraries (default: polygeist stack)",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		names := args
		if len(names) == 0 {
			names = []string{"polygeist"}
		}

		for _, name := range names {
			if strings.EqualFold(name, "polygeist") {
				if err := ecosystem.InstallPolygeistStack(); err != nil {
					fmt.Fprintf(os.Stderr, "install polygeist: %v\n", err)
					os.Exit(1)
				}
				continue
			}
			if err := ecosystem.Install(name); err != nil {
				fmt.Fprintf(os.Stderr, "install %s: %v\n", name, err)
				os.Exit(1)
			}
		}
	},
}

func printLibraries(libs []ecosystem.Library, title string) {
	fmt.Printf("\n%s:\n", title)
	for _, lib := range libs {
		status := "missing"
		if ecosystem.IsInstalled(lib) {
			status = "installed"
		}
		fmt.Printf("  %-12s [%s]  %s\n", lib.Name, status, lib.Description)
	}
}

func init() {
	stackListCmd.Flags().StringVar(&stackCategory, "category", "", "Filter by category (internal, tool)")
	stackCmd.AddCommand(stackListCmd)
	stackCmd.AddCommand(stackInstallCmd)
	rootCmd.AddCommand(stackCmd)
}
