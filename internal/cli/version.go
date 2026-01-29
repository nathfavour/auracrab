package cli

import (
	"fmt"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	versionCmd.Flags().Bool("short-commit", false, "Print only the short commit SHA")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of auracrab",
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short-commit")
		if short {
			fmt.Print(config.Commit)
			return
		}
		fmt.Println(rootCmd.Version)
	},
}
