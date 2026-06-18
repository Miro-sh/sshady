package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SSH configs managed by sshady",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := sshconf.ReadManagedEntries()
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			configPath, err := sshconf.ConfigFilePath()
	if err != nil {
		configPath = "~/.ssh/config"
	}
			fmt.Printf("No entries managed by sshady in %s.
", configPath)
			fmt.Println("Run 'sshady create' to add one.")
			return nil
		}

		configPath, err := sshconf.ConfigFilePath()
	if err != nil {
		configPath = "~/.ssh/config"
	}
		fmt.Printf("Managed entries in %s:

", configPath)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ALIAS	HOST	USER	PORT	PROXY")
		fmt.Fprintln(w, "-----	----	----	----	-----")
		for _, e := range entries {
			fmt.Fprintf(w, "%s	%s	%s	%s	%s
",
				e.Alias, e.HostName, e.User, e.Port, e.ProxySummary)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
