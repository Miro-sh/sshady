package cmd

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"sshady/internal/sshconf"
)

var testTimeout int

var testCmd = &cobra.Command{
	Use:   "test <alias>",
	Short: "Test that the proxy for an sshady-managed entry is reachable",
	Long: `Attempt a TCP connection to the proxy server configured for the given alias.
Uses ncat -z (connect-only mode) to verify the proxy is reachable.

Exit code 0: proxy is reachable.
Exit code 1: proxy is unreachable or the alias is not managed by sshady.`,
	Example: `  sshady test myserver
  sshady test myserver --timeout 10
  sshady test hidden --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		entry, err := sshconf.FindEntry(alias)
		if err != nil {
			return err
		}
		if entry == nil {
			return fmt.Errorf("alias %q is not managed by sshady; use 'sshady list' to see managed entries", alias)
		}

		fmt.Printf("Testing proxy for %q...
", alias)
		fmt.Printf("  Proxy: %s
", entry.ProxySummary)

		addr := extractAddr(entry.ProxySummary)
		if addr == "" && entry.ProxySummary != "Tor (127.0.0.1:9050)" {
			fmt.Println("  Cannot determine proxy address from summary — skipping test")
			return nil
		}
		if addr == "" {
			addr = "127.0.0.1:9050"
		}

		host, port, err := splitHostPort(addr)
		if err != nil {
			return fmt.Errorf("cannot parse proxy address %q: %w", addr, err)
		}

		if verboseMode {
			fmt.Printf("  Connecting to %s:%s (timeout: %ds)...
", host, port, testTimeout)
		} else {
			fmt.Printf("  Connecting to %s:%s...
", host, port)
		}

		timeout := time.Duration(testTimeout) * time.Second
		ncatCmd := exec.Command("ncat", "-z", "-w", strconv.Itoa(testTimeout), host, port)

		start := time.Now()
		output, err := ncatCmd.CombinedOutput()
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("  ✗ Proxy unreachable after %v: %v
", elapsed.Round(time.Millisecond), err)
			if verboseMode && len(output) > 0 {
				fmt.Printf("  ncat output: %s
", string(output))
			}
			return fmt.Errorf("proxy test failed")
		}

		fmt.Printf("  ✓ Proxy reachable (%v)
", elapsed.Round(time.Millisecond))
		return nil
	},
}

// extractAddr extracts "host:port" from a proxy summary string.
// Handles formats: "SOCKS5 via host:port", "HTTP via host:port", "SSH Jump -> user@host"
func extractAddr(summary string) string {
	// For "via host:port" format
	if idx := strings.LastIndex(summary, " via "); idx >= 0 {
		return summary[idx+5:]
	}
	// For "Jump -> user@host" — extract just the host part
	if idx := strings.LastIndex(summary, " -> "); idx >= 0 {
		rest := summary[idx+4:]
		if atIdx := strings.LastIndex(rest, "@"); atIdx >= 0 {
			return rest[atIdx+1:]
		}
		return rest
	}
	return ""
}

// splitHostPort splits "host:port" safely, handling IPv6 addresses like [::1]:1080.
func splitHostPort(addr string) (host, port string, err error) {
	// Try net.SplitHostPort first (handles IPv6 correctly)
	h, p, err := net.SplitHostPort(addr)
	if err == nil {
		return h, p, nil
	}
	// Fallback: assume host without port
	return addr, "", fmt.Errorf("address %q does not contain a valid host:port", addr)
}

func init() {
	testCmd.Flags().IntVar(&testTimeout, "timeout", 5, "Connection timeout in seconds")
	rootCmd.AddCommand(testCmd)
}
