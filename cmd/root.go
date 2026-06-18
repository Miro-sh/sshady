// Package cmd implements the sshady CLI command tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

// Version is set at build time via ldflags.
var Version = "dev"

var (
	dryRun      bool
	forceWrite  bool
	quietMode   bool
	verboseMode bool
	configPath  string
)

var rootCmd = &cobra.Command{
	Use:   "sshady",
	Short: "SSH proxy config generator — hide your IP, look innocent",
	Long: `sshady generates SSH Host entries with proxy configurations
and writes them directly to ~/.ssh/config.

Supports SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.
Run without arguments for the interactive wizard.`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: `  # Launch the interactive wizard
  sshady

  # Create a proxy config (non-interactive)
  sshady create --alias myserver --host 1.2.3.4 --user admin --proxy-type socks5 --proxy-host proxy.example.com

  # List all managed entries
  sshady list

  # Show details
  sshady show myserver

  # Validate config
  sshady validate myserver --test-proxy

  # Test proxy reachability
  sshady test myserver

  # Remove an entry
  sshady delete myserver

  # Generate shell completion
  source <(sshady completion bash)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWizard()
	},
}

// SetVersion allows setting the version from main.go.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false,
		"Preview changes without modifying the config file")
	rootCmd.PersistentFlags().BoolVar(&forceWrite, "force", false,
		"Overwrite existing sshady-managed entries without confirmation")
	rootCmd.PersistentFlags().BoolVarP(&quietMode, "quiet", "q", false,
		"Suppress non-error output (useful for scripting)")
	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false,
		"Enable verbose diagnostic output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "",
		"Path to SSH config file (default: ~/.ssh/config)")

	cobra.OnInitialize(func() {
		if configPath != "" {
			sshconf.SetConfigPath(configPath)
		}
	})
}

// Execute runs the root command. Exits with code 1 on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
