// sshady — SSH proxy config generator.
//
// Generate SSH Host entries with proxy configurations and write them
// directly to ~/.ssh/config. Supports SOCKS5, HTTP CONNECT, Tor, and
// SSH jump hosts.
//
// Build:
//   make build
//   go build -ldflags "-X sshady/cmd.Version=v1.0.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
package main

import "sshady/cmd"

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
