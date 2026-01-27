package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of auracrab",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("auracrab %s\n", Version)
	},
}
