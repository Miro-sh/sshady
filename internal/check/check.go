// Package check provides runtime environment checks for prerequisites
// like ncat availability and Tor daemon status.
package check

import (
	"fmt"
	"os/exec"
	"strings"
)

// NcatVersion returns the installed ncat version string, or an error if not found.
func NcatVersion() (string, error) {
	out, err := exec.Command("ncat", "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ncat not found or not executable: %w", err)
	}
	// Output format: "Ncat: Version 7.94 ..."
	line := strings.SplitN(string(out), "
", 2)[0]
	return strings.TrimSpace(line), nil
}

// HasNcat returns true if ncat is available in PATH.
func HasNcat() bool {
	_, err := NcatVersion()
	return err == nil
}

// TorRunning checks if a Tor SOCKS port is accepting connections on the given address.
// Default Tor SOCKS port is 127.0.0.1:9050.
func TorRunning(addr string) (bool, error) {
	if addr == "" {
		addr = "127.0.0.1:9050"
	}
	// Use ncat -z for a quick TCP connect test
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		return false, fmt.Errorf("invalid Tor address %q", addr)
	}
	cmd := exec.Command("ncat", "-z", "-w", "2", host, port)
	err := cmd.Run()
	return err == nil, nil
}

// CheckPrerequisites verifies all runtime dependencies.
// Returns a list of issues found (empty = all good).
func CheckPrerequisites() []string {
	var issues []string

	if !HasNcat() {
		issues = append(issues, "ncat not found — install the 'nmap' package")
	}

	// Check that ~/.ssh directory exists with correct permissions is done by sshconf

	return issues
}
