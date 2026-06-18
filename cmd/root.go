// Package cmd implements the sshady CLI command tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

// Version is set at build time via ldflags:
//   go build -ldflags "-X sshady/cmd.Version=v1.0.0"
var Version = "dev"

var (
	// dryRun controls whether write operations are skipped.
	dryRun bool
	// forceWrite allows overwriting existing entries.
	forceWrite bool
	// configPath is an alternative SSH config path.
	configPath string
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
	// Silence the default usage and error output — we handle errors ourselves.
	SilenceUsage:  true,
	SilenceErrors: true,
}

// SetVersion allows setting the version from main.go.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be written without modifying the config file")
	rootCmd.PersistentFlags().BoolVar(&forceWrite, "force", false, "Overwrite existing sshady-managed entries without confirmation")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to SSH config file (default: ~/.ssh/config)")

	// Apply --config override before any command runs
	cobra.OnInitialize(func() {
		if configPath != "" {
			sshconf.SetConfigPath(configPath)
		}
	})
}

// Execute runs the root command. Exits with code 1 on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
}
