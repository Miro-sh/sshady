package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all SSH configs managed by sshady",
	Example: `  sshady list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := sshconf.ReadManagedEntries()
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("No entries managed by sshady yet. Run 'sshady create' to add one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ALIAS\tHOST\tUSER\tPORT\tPROXY")
		fmt.Fprintln(w, "-----\t----\t----\t----\t-----")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				e.Alias, e.HostName, e.User, e.Port, e.ProxySummary)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
