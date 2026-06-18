package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <alias>",
	Short: "Remove an SSH config managed by sshady",
	Long: `Remove an sshady-managed entry from the SSH config file by its alias.

A timestamped backup is automatically created before removal.
Use --dry-run to preview without modifying the config.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		if dryRun {
			fmt.Printf("[DRY RUN] Would remove alias %q from SSH config
", alias)
			return nil
		}

		if err := sshconf.RemoveEntry(alias); err != nil {
			return err
		}

		configPath, err := sshconf.ConfigFilePath()
	if err != nil {
		configPath = "~/.ssh/config"
	}
		fmt.Printf("✓ Removed 'Host %s' from %s
", alias, configPath)
		fmt.Println("  A timestamped backup was saved.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
