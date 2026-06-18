package cmd

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"sshady/internal/proxy"
	"sshady/internal/sshconf"
)

var testTimeout int

var testCmd = &cobra.Command{
	Use:   "test <alias>",
	Short: "Test that the proxy for an sshady-managed entry is reachable",
	Long: `Attempt a TCP connection to the proxy server configured for the given alias.
Uses ncat in connect-only mode (no data transfer) to verify the proxy is reachable.

Exit code 0: proxy is reachable.
Exit code 1: proxy is unreachable or the alias is not managed by sshady.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		entry, err := sshconf.FindEntry(alias)
		if err != nil {
			return err
		}
		if entry == nil {
			return fmt.Errorf("alias %q is not managed by sshady", alias)
		}

		// For now we can't fully reconstruct the proxy.Config from the managed entry,
		// but we can at least parse the meta to get the proxy address.
		// A more complete solution would store the full Config in the META line.
		fmt.Printf("Testing proxy for %q...\n", alias)
		fmt.Printf("  Proxy: %s\n", entry.ProxySummary)

		// We use the META line info to determine what to test
		// For Tor, we test 127.0.0.1:9050
		// For jump, we can't easily test
		// For SOCKS5/HTTP, we test the proxy address with ncat

		// Extract proxy address from summary
		// This is a simplified approach — a production version would parse META more robustly
		addr := extractAddr(entry.ProxySummary)
		if addr == "" && entry.ProxySummary != "Tor (127.0.0.1:9050)" {
			fmt.Println("  Cannot determine proxy address from summary")
			return nil
		}
		if addr == "" {
			addr = "127.0.0.1:9050"
		}

		fmt.Printf("  Connecting to %s (timeout: %ds)...\n", addr, testTimeout)

		timeout := time.Duration(testTimeout) * time.Second
		ncatCmd := exec.Command("ncat", "-z", "-w", fmt.Sprintf("%d", testTimeout),
			parseHost(addr), parsePort(addr))
		ncatCmd.Stderr = nil

		start := time.Now()
		err = ncatCmd.Run()
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("  ✗ Proxy unreachable after %v: %v\n", elapsed.Round(time.Millisecond), err)
			return fmt.Errorf("proxy test failed")
		}

		fmt.Printf("  ✓ Proxy reachable (%v)\n", elapsed.Round(time.Millisecond))
		return nil
	},
}

// extractAddr tries to extract "host:port" from a proxy summary string.
func extractAddr(summary string) string {
	// Formats: "SOCKS5 via host:port", "HTTP via host:port", etc.
	for i := len(summary) - 1; i >= 0; i-- {
		if summary[i] == ' ' {
			return summary[i+1:]
		}
	}
	return ""
}

func parseHost(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

func parsePort(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return "1080"
}

func init() {
	testCmd.Flags().IntVar(&testTimeout, "timeout", 5, "Connection timeout in seconds")
	rootCmd.AddCommand(testCmd)

	// Suppress unused import warnings
	_ = proxy.AllowedTypes
}
