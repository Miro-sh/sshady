package main

import "sshady/cmd"

// These are set via ldflags at build time:
//   go build -ldflags "-X sshady/cmd.Version=v1.0.0"
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
