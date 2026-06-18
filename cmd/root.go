package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var (
	// dryRun controls whether write operations are skipped.
	dryRun bool
)

var rootCmd = &cobra.Command{
	Use:   "sshady",
	Short: "SSH proxy config generator — hide your IP, look innocent",
	Long: `sshady generates SSH Host entries with proxy configurations
and writes them directly to ~/.ssh/config.

Supports SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.
Run without arguments for the interactive wizard.`,
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWizard()
	},
}

// SetVersion allows setting the version from main.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be written without modifying ~/.ssh/config")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
}
