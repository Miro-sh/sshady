package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:     "delete <alias>",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete an SSH proxy configuration",
	Long: `Delete an entry managed by sshady from ~/.ssh/config.

Asks for confirmation unless --force is given.
A backup is written to ~/.ssh/config.sshady.bak before every change.`,
	Example: `  sshady delete myserver
  sshady delete myserver --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		if _, err := sshconf.ReadEntry(alias); err != nil {
			return err
		}

		if !deleteForce {
			var confirm bool
			if err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Delete 'Host %s' from ~/.ssh/config?", alias),
				Default: false,
			}, &confirm); err != nil || !confirm {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := sshconf.DeleteEntry(alias); err != nil {
			return err
		}

		fmt.Printf("Deleted 'Host %s' from ~/.ssh/config\n", alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "delete without confirmation")
}
