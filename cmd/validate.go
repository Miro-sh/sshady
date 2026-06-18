package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"sshady/internal/sshconf"
)

var validateCmd = &cobra.Command{
	Use:   "validate <alias>",
	Short: "Validate an sshady-managed config entry",
	Long: `Check that an sshady-managed SSH config entry is valid.

Runs 'ssh -G <alias>' to validate the SSH config syntax,
then optionally tests proxy reachability.

Exit code 0: config is valid.
Exit code 1: config has issues.`,
	Example: `  sshady validate myserver
  sshady validate myserver --test-proxy
  sshady validate --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if validateAll {
			return validateAllEntries()
		}
		if len(args) == 0 {
			return fmt.Errorf("specify an alias or use --all to validate all entries")
		}
		return validateOne(args[0])
	},
}

var (
	validateAll   bool
	validateProxy bool
)

func init() {
	validateCmd.Flags().BoolVar(&validateAll, "all", false, "Validate all sshady-managed entries")
	validateCmd.Flags().BoolVar(&validateProxy, "test-proxy", false, "Also test proxy reachability")
	rootCmd.AddCommand(validateCmd)
}

func validateOne(alias string) error {
	entry, err := sshconf.FindEntry(alias)
	if err != nil {
		return err
	}
	if entry == nil {
		return fmt.Errorf("alias %q is not managed by sshady", alias)
	}

	fmt.Printf("Validating %q...
", alias)

	// Run ssh -G to validate SSH config parsing
	sshCmd := exec.Command("ssh", "-G", alias)
	output, err := sshCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  ✗ SSH config validation failed for %q:
", alias)
		fmt.Printf("  %s
", string(output))
		return fmt.Errorf("ssh -G %s failed: %w", alias, err)
	}

	fmt.Printf("  ✓ SSH config syntax: valid
")
	fmt.Printf("  HostName: %s | User: %s | Port: %s | Proxy: %s
",
		entry.HostName, entry.User, entry.Port, entry.ProxySummary)

	if validateProxy {
		fmt.Println("  Testing proxy reachability...")
		// Delegate to test command logic
		testCmd.RunE(cmd, []string{alias})
	}

	return nil
}

func validateAllEntries() error {
	entries, err := sshconf.ReadManagedEntries()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No entries managed by sshady.")
		return nil
	}

	failures := 0
	for _, e := range entries {
		fmt.Printf("
--- %s ---
", e.Alias)
		if err := validateOne(e.Alias); err != nil {
			failures++
			if !verboseMode {
				fmt.Printf("  ✗ Validation failed: %v
", err)
			}
		}
	}

	fmt.Printf("
Results: %d/%d passed
", len(entries)-failures, len(entries))
	if failures > 0 {
		return fmt.Errorf("%d validation(s) failed", failures)
	}
	return nil
}
