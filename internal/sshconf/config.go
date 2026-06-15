package sshconf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sshady/internal/proxy"
)

type HostConfig struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	Proxy        proxy.Config
}

type ManagedEntry struct {
	Alias     string
	HostName  string
	User      string
	Port      string
	ProxySummary string
}

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

func WriteEntry(cfg HostConfig) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	if err := ensureSSHDir(); err != nil {
		return err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return err
	}

	if strings.Contains(content, fmt.Sprintf("# BEGIN SSHADY:%s", cfg.Alias)) {
		return fmt.Errorf("alias %q is already managed by sshady; use 'sshady list' to see existing entries", cfg.Alias)
	}

	re := regexp.MustCompile(fmt.Sprintf(`(?m)^Host\s+%s(\s|$)`, regexp.QuoteMeta(cfg.Alias)))
	if re.MatchString(content) {
		return fmt.Errorf("alias %q already exists in ~/.ssh/config (not managed by sshady)", cfg.Alias)
	}

	if content != "" {
		if err := os.WriteFile(configPath+".sshady.bak", []byte(content), 0600); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	newContent := content
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	if newContent != "" {
		newContent += "\n"
	}
	newContent += cfg.Block()

	return atomicWrite(configPath, newContent, 0600)
}

func ReadManagedEntries() ([]ManagedEntry, error) {
	configPath, err := configFilePath()
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
		if strings.HasPrefix(line, "# BEGIN SSHADY:") {
			inBlock = true
			current = ManagedEntry{Alias: strings.TrimPrefix(line, "# BEGIN SSHADY:")}
		} else if strings.HasPrefix(line, "# END SSHADY:") {
			if inBlock {
				entries = append(entries, current)
			}
			inBlock = false
		} else if inBlock {
			switch {
			case strings.HasPrefix(trimmed, "HostName "):
				current.HostName = strings.TrimPrefix(trimmed, "HostName ")
			case strings.HasPrefix(trimmed, "User "):
				current.User = strings.TrimPrefix(trimmed, "User ")
			case strings.HasPrefix(trimmed, "Port "):
				current.Port = strings.TrimPrefix(trimmed, "Port ")
			case strings.HasPrefix(line, "# SSHADY META:"):
				current.ProxySummary = parseMetaSummary(strings.TrimPrefix(line, "# SSHADY META: "))
			}
		}
	}

	return entries, nil
}

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

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

func ensureSSHDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(home, ".ssh"), 0700)
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

func atomicWrite(path, content string, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".sshady-tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
