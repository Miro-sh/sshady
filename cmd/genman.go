package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var genManCmd = &cobra.Command{
	Use:    "gen-man [output-dir]",
	Short:  "Generate man pages (maintainer tool)",
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "man"
		if len(args) > 0 {
			dir = args[0]
		}

		header := &doc.GenManHeader{
			Title:   "SSHADY",
			Section: "1",
			Source:  "sshady " + version,
			Manual:  "sshady Manual",
		}

		if err := doc.GenManTree(rootCmd, header, dir); err != nil {
			return err
		}

		fmt.Printf("Man pages written to %s/\n", dir)
		fmt.Printf("Install with:  sudo cp %s/*.1 /usr/local/share/man/man1/ && sudo mandb\n", dir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(genManCmd)
}
