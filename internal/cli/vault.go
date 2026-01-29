package cli

import (
	"fmt"

	"github.com/nathfavour/auracrab/pkg/vault"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage secure secrets in the OS keychain",
}

var vaultSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a secret in the vault",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		v := vault.GetVault()
		if err := v.Set(args[0], args[1]); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Secret '%s' set successfully.\n", args[0])
	},
}

var revealVault bool

var vaultGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a secret from the vault",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		v := vault.GetVault()
		val, err := v.Get(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if revealVault {
			fmt.Printf("%s: %s\n", args[0], val)
		} else {
			fmt.Printf("%s: %s\n", args[0], vault.Mask(val))
		}
	},
}

func init() {
	vaultGetCmd.Flags().BoolVarP(&revealVault, "reveal", "r", false, "Reveal the secret value")
	vaultCmd.AddCommand(vaultSetCmd)
	vaultCmd.AddCommand(vaultGetCmd)
	rootCmd.AddCommand(vaultCmd)
}
