package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "sshady",
	Version: version,
	Short:   "SSH proxy config generator — hide your IP, look innocent",
	Long: `sshady generates SSH Host entries with proxy configurations
and writes them directly to ~/.ssh/config.

Supports SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.
Run without arguments for the interactive wizard.`,
	Example: `  sshady                                    Launch the interactive wizard
  sshady create --alias web --host 1.2.3.4 --proxy-type tor
  sshady list                               List managed entries
  sshady show web                           Show entry details
  sshady edit web --port 443                Update one field
  sshady delete web                         Remove an entry`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWizard()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
