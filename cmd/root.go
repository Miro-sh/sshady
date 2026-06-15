package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sshady",
	Short: "SSH proxy config generator — hide your IP, look innocent",
	Long: `sshady generates SSH Host entries with proxy configurations
and writes them directly to ~/.ssh/config.

Supports SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.
Run without arguments for the interactive wizard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWizard()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
