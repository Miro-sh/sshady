// Package sshconf manages reading, writing, and parsing of ~/.ssh/config
// with security-first design: atomic writes, backup rotation, input validation,
// and strict permissions.
package sshconf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"sshady/internal/proxy"
)

// ── Types ─────────────────────────────────────────────────

// HostConfig holds the full configuration for an SSH host entry.
type HostConfig struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	Proxy        proxy.Config
}

// ManagedEntry is a lightweight view of a managed host for listing.
type ManagedEntry struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	ProxySummary string
}

// ── Configuration path ────────────────────────────────────

// configPathOverride allows tests and --config flag to override the SSH config path.
var configPathOverride string

// SetConfigPath overrides the default ~/.ssh/config path.
// Pass empty string to reset to default.
func SetConfigPath(path string) {
	configPathOverride = path
}

// ConfigFilePath returns the path to the SSH config file.
func ConfigFilePath() (string, error) {
	if configPathOverride != "" {
		return configPathOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// backupPath returns the path for a backup file with optional timestamp.
func backupPath(configPath string, timestamped bool) string {
	if timestamped {
		ts := time.Now().UTC().Format("20060102-150405")
		return configPath + ".sshady." + ts + ".bak"
	}
	return configPath + ".sshady.bak"
}

// ── Validation ────────────────────────────────────────────

// ValidateHostConfig checks the entire HostConfig for safety.
func ValidateHostConfig(cfg *HostConfig) error {
	if err := proxy.ValidateAlias(cfg.Alias); err != nil {
		return err
	}
	if cfg.HostName == "" {
		return fmt.Errorf("target host is required; use --host")
	}
	if err := proxy.ValidateHost(cfg.HostName); err != nil {
		return fmt.Errorf("target host: %w", err)
	}
	if cfg.User == "" {
		return fmt.Errorf("SSH user is required; use --user")
	}
	if err := proxy.ValidateUserPass(cfg.User, "SSH user"); err != nil {
		return fmt.Errorf("SSH user: %w", err)
	}
	if cfg.Port == "" {
		cfg.Port = "22"
	}
	if err := proxy.ValidatePort(cfg.Port); err != nil {
		return fmt.Errorf("SSH port: %w", err)
	}
	if cfg.IdentityFile != "" {
		if strings.ContainsAny(cfg.IdentityFile, "\n\r") {
			return fmt.Errorf("identity file path contains invalid characters")
		}
	}
	if err := cfg.Proxy.Validate(); err != nil {
		return fmt.Errorf("proxy config: %w", err)
	}
	return nil
}

// ── Block generation ──────────────────────────────────────

// Block generates the SSH config block for a HostConfig.
func (h HostConfig) Block() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# BEGIN SSHADY:%s\n", h.Alias)
	fmt.Fprintf(&b, "# SSHADY META: %s\n", h.Proxy.MetaString())
	fmt.Fprintf(&b, "Host %s\n", h.Alias)
	fmt.Fprintf(&b, "    HostName %s\n", h.HostName)
	fmt.Fprintf(&b, "    User %s\n", h.User)
	fmt.Fprintf(&b, "    Port %s\n", h.Port)
	if h.IdentityFile != "" {
		fmt.Fprintf(&b, "    IdentityFile %s\n", h.IdentityFile)
	}
	if h.Proxy.Type == proxy.TypeJump {
		fmt.Fprintf(&b, "    ProxyJump %s\n", h.Proxy.JumpHost)
	} else {
		fmt.Fprintf(&b, "    ProxyCommand %s\n", h.Proxy.ProxyCommand())
	}
	fmt.Fprintf(&b, "# END SSHADY:%s\n", h.Alias)
	return b.String()
}

// ── Write / Remove operations ─────────────────────────────

// WriteEntry writes a host entry to the SSH config file.
// If force is true, overwrites an existing sshady-managed entry with the same alias.
func WriteEntry(cfg HostConfig, force bool) error {
	if err := ValidateHostConfig(&cfg); err != nil {
		return err
	}

	configPath, err := ConfigFilePath()
	if err != nil {
		return err
	}

	if err := ensureSSHDir(configPath); err != nil {
		return err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return err
	}

	// Check for existing sshady-managed entry with same alias
	beginMarker := fmt.Sprintf("# BEGIN SSHADY:%s", cfg.Alias)
	if strings.Contains(content, beginMarker) {
		if !force {
			return fmt.Errorf("alias %q is already managed by sshady; use --force to overwrite, or 'sshady delete %s' first", cfg.Alias, cfg.Alias)
		}
		// Remove the existing block before adding the new one
		content = removeBlock(content, cfg.Alias)
	}

	// Check for conflicting non-sshady Host entry
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^Host\s+%s(\s|$)`, regexp.QuoteMeta(cfg.Alias)))
	if re.MatchString(content) {
		return fmt.Errorf("alias %q already exists in ~/.ssh/config (not managed by sshady); remove it manually or choose a different alias", cfg.Alias)
	}

	// Create timestamped backup
	if content != "" {
		if err := createBackup(configPath, content); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		if err := rotateBackups(configPath, 5); err != nil {
			// Non-fatal: log but don't fail the write
			fmt.Fprintf(os.Stderr, "Warning: backup rotation failed: %v\n", err)
		}
	}

	// Build new content
	newContent := strings.TrimRight(content, "\n")
	if newContent != "" {
		newContent += "\n\n"
	}
	newContent += cfg.Block()
	newContent += "\n"

	return atomicWrite(configPath, newContent, 0600)
}

// RemoveEntry removes an sshady-managed entry by alias from the SSH config.
func RemoveEntry(alias string) error {
	configPath, err := ConfigFilePath()
	if err != nil {
		return err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return err
	}

	beginMarker := fmt.Sprintf("# BEGIN SSHADY:%s", alias)
	if !strings.Contains(content, beginMarker) {
		return fmt.Errorf("alias %q is not managed by sshady; use 'sshady list' to see managed entries", alias)
	}

	// Create backup before modifying
	if err := createBackup(configPath, content); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	newContent := removeBlock(content, alias)
	if newContent != "" {
		newContent += "\n"
	}

	return atomicWrite(configPath, newContent, 0600)
}

// removeBlock strips the BEGIN/END SSHADY block for the given alias from content.
func removeBlock(content, alias string) string {
	beginMarker := fmt.Sprintf("# BEGIN SSHADY:%s", alias)
	endMarker := fmt.Sprintf("# END SSHADY:%s", alias)

	lines := strings.Split(content, "\n")
	var result []string
	skip := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == beginMarker {
			skip = true
			// Remove preceding blank lines
			for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
				result = result[:len(result)-1]
			}
			continue
		}
		if trimmed == endMarker {
			skip = false
			continue
		}
		if skip {
			continue
		}
		result = append(result, line)
	}

	// Trim trailing blank lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n")
}

// ── Reading ───────────────────────────────────────────────

// ReadManagedEntries reads all sshady-managed entries from the SSH config.
func ReadManagedEntries() ([]ManagedEntry, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return nil, err
	}

	var entries []ManagedEntry
	lines := strings.Split(content, "\n")
	inBlock := false
	var current ManagedEntry

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# BEGIN SSHADY:"):
			inBlock = true
			current = ManagedEntry{Alias: strings.TrimPrefix(trimmed, "# BEGIN SSHADY:")}
		case strings.HasPrefix(trimmed, "# END SSHADY:"):
			if inBlock {
				entries = append(entries, current)
			}
			inBlock = false
		case inBlock:
			switch {
			case strings.HasPrefix(trimmed, "HostName "):
				current.HostName = strings.TrimPrefix(trimmed, "HostName ")
			case strings.HasPrefix(trimmed, "User "):
				current.User = strings.TrimPrefix(trimmed, "User ")
			case strings.HasPrefix(trimmed, "Port "):
				current.Port = strings.TrimPrefix(trimmed, "Port ")
			case strings.HasPrefix(trimmed, "# SSHADY META:"):
				meta := strings.TrimPrefix(trimmed, "# SSHADY META: ")
				current.ProxySummary = parseMetaSummary(meta)
			}
		}
	}

	return entries, nil
}

// FindEntry finds a single managed entry by alias. Returns nil if not found.
func FindEntry(alias string) (*ManagedEntry, error) {
	entries, err := ReadManagedEntries()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Alias == alias {
			return &entries[i], nil
		}
	}
	return nil, nil
}

// parseMetaSummary converts a META line into a human-readable proxy summary.
func parseMetaSummary(meta string) string {
	fields := map[string]string{}
	for _, part := range strings.Fields(meta) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			fields[kv[0]] = kv[1]
		}
	}

	switch fields["type"] {
	case "socks5":
		s := fmt.Sprintf("SOCKS5 via %s", fields["addr"])
		if fields["auth"] == "yes" {
			s += " (auth)"
		}
		return s
	case "socks4":
		return fmt.Sprintf("SOCKS4 via %s", fields["addr"])
	case "socks4a":
		return fmt.Sprintf("SOCKS4a via %s", fields["addr"])
	case "http":
		s := fmt.Sprintf("HTTP via %s", fields["addr"])
		if fields["auth"] == "yes" {
			s += " (auth)"
		}
		return s
	case "tor":
		return "Tor (127.0.0.1:9050)"
	case "jump":
		return fmt.Sprintf("SSH Jump -> %s", fields["host"])
	}
	return meta
}

// ── Filesystem helpers ────────────────────────────────────

func ensureSSHDir(configPath string) error {
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0700)
}

func readConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	return string(data), nil
}

// createBackup writes content to a timestamped backup file with 0600 permissions.
func createBackup(configPath, content string) error {
	bp := backupPath(configPath, true)
	return os.WriteFile(bp, []byte(content), 0600)
}

// rotateBackups keeps only the most recent N timestamped backup files.
func rotateBackups(configPath string, keep int) error {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	prefix := base + ".sshady."
	var backups []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".bak") {
			backups = append(backups, e.Name())
		}
	}

	if len(backups) <= keep {
		return nil
	}

	// Sort by name (which includes timestamp) — oldest first
	sort.Strings(backups)
	toDelete := len(backups) - keep
	for i := 0; i < toDelete; i++ {
		if err := os.Remove(filepath.Join(dir, backups[i])); err != nil {
			fmt.Fprintf(os.Stderr, "sshady: warning: could not remove old backup %s: %v
", backups[i], err)
		}
	}

	return nil
}

// ── Atomic write ──────────────────────────────────────────

// atomicWrite writes content to path by writing to a temp file, fsyncing, and renaming.
// This ensures the target file is never partially written.
// The temp file is created in the same directory as the target to guarantee
// the rename is atomic (same filesystem).
// On success, the temp file is renamed — the deferred Remove is a no-op.
// On error, the deferred Remove cleans up the temp file.
func atomicWrite(path, content string, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".sshady-tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on error paths.
	// On success, the file is renamed away so os.Remove is a no-op (file not found).
	defer func() {
		os.Remove(tmpName)
	}()

	// Set permissions before writing
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	// Ensure durability: sync to disk before rename
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename (same filesystem, so it's truly atomic on POSIX)
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	return nil
}
