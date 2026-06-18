package check

import (
	"testing"
)

func TestHasNcat(t *testing.T) {
	// This test verifies the function doesn't panic.
	// We can't guarantee ncat is installed in CI, so we just check it runs.
	_ = HasNcat()
}

func TestTorRunning(t *testing.T) {
	// Don't assume Tor is running; just check the function doesn't panic.
	_, err := TorRunning("127.0.0.1:9050")
	// Error is expected if Tor isn't running, but the function shouldn't crash.
	_ = err
}

func TestCheckPrerequisites(t *testing.T) {
	issues := CheckPrerequisites()
	// Just verify it runs without panic
	t.Logf("Prerequisite issues: %v", issues)
}
