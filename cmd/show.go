package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var showReveal bool

var showCmd = &cobra.Command{
	Use:   "show <alias>",
	Short: "Show full details of an SSH proxy configuration",
	Long: `Show every field of an entry managed by sshady, including the
exact block written to ~/.ssh/config.

Proxy passwords are masked unless --reveal is given.`,
	Example: `  sshady show myserver
  sshady show myserver --reveal`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		cfg, err := sshconf.ReadEntry(alias)
		if err != nil {
			return err
		}

		fmt.Printf("Alias:     %s\n", cfg.Alias)
		fmt.Printf("HostName:  %s\n", cfg.HostName)
		fmt.Printf("User:      %s\n", cfg.User)
		fmt.Printf("Port:      %s\n", cfg.Port)
		if cfg.IdentityFile != "" {
			fmt.Printf("Identity:  %s\n", cfg.IdentityFile)
		}
		fmt.Printf("Proxy:     %s\n", cfg.Proxy.Summary())
		fmt.Println()
		fmt.Println("~/.ssh/config block:")
		fmt.Println()

		block := cfg.Block()
		if !showReveal {
			block = maskProxyAuth(block)
		}
		fmt.Print(indent(block))

		return nil
	},
}

func maskProxyAuth(s string) string {
	re := regexp.MustCompile(`(--proxy-auth [^:\s]+):[^\s]+`)
	return re.ReplaceAllString(s, "${1}:********")
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n") + "\n"
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showReveal, "reveal", false, "show proxy password in plaintext")
}
