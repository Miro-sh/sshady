package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var showCmd = &cobra.Command{
	Use:   "show <alias>",
	Short: "Show details of an sshady-managed entry",
	Long: `Display the full configuration block that sshady wrote for the given alias,
including proxy type, address, and authentication status.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		entry, err := sshconf.FindEntry(alias)
		if err != nil {
			return err
		}
		if entry == nil {
			return fmt.Errorf("alias %q is not managed by sshady; use 'sshady list' to see managed entries", alias)
		}

		fmt.Printf("Alias:       %s
", entry.Alias)
		fmt.Printf("HostName:    %s
", entry.HostName)
		fmt.Printf("User:        %s
", entry.User)
		fmt.Printf("Port:        %s
", entry.Port)
		fmt.Printf("Proxy:       %s
", entry.ProxySummary)
		fmt.Println()
		fmt.Println("Connect with:  ssh", entry.Alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
