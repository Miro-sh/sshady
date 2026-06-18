package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <alias>",
	Short: "Remove an SSH config managed by sshady",
	Long: `Remove an sshady-managed entry from ~/.ssh/config by its alias.
A backup is automatically created at ~/.ssh/config.sshady.bak.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		if dryRun {
			fmt.Printf("[DRY RUN] Would remove alias %q from ~/.ssh/config
", alias)
			return nil
		}

		if err := sshconf.RemoveEntry(alias); err != nil {
			return err
		}

		fmt.Printf("Removed 'Host %s' from ~/.ssh/config
", alias)
		fmt.Println("A backup was saved to ~/.ssh/config.sshady.bak")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
