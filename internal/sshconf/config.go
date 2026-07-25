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

	if strings.Contains(content, fmt.Sprintf("# BEGIN SSHADY:%s\n", cfg.Alias)) {
		return fmt.Errorf("alias %q is already managed by sshady; use 'sshady list' to see existing entries", cfg.Alias)
	}

	if err := checkAliasFree(content, cfg.Alias); err != nil {
		return err
	}

	if err := backup(configPath, content); err != nil {
		return err
	}

	return atomicWrite(configPath, appendBlock(content, cfg.Block()), 0600)
}

func ReadEntry(alias string) (HostConfig, error) {
	configPath, err := configFilePath()
	if err != nil {
		return HostConfig{}, err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return HostConfig{}, err
	}

	cfg := HostConfig{Alias: alias}
	meta := ""
	inBlock := false
	found := false

	for _, line := range strings.Split(content, "\n") {
		if line == "# BEGIN SSHADY:"+alias {
			inBlock = true
			found = true
			continue
		}
		if line == "# END SSHADY:"+alias {
			break
		}
		if !inBlock {
			continue
		}

		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "# SSHADY META:"):
			meta = strings.TrimPrefix(line, "# SSHADY META: ")
		case strings.HasPrefix(trimmed, "HostName "):
			cfg.HostName = strings.TrimPrefix(trimmed, "HostName ")
		case strings.HasPrefix(trimmed, "User "):
			cfg.User = strings.TrimPrefix(trimmed, "User ")
		case strings.HasPrefix(trimmed, "Port "):
			cfg.Port = strings.TrimPrefix(trimmed, "Port ")
		case strings.HasPrefix(trimmed, "IdentityFile "):
			cfg.IdentityFile = strings.TrimPrefix(trimmed, "IdentityFile ")
		case strings.HasPrefix(trimmed, "ProxyJump "):
			cfg.Proxy.Type = proxy.TypeJump
			cfg.Proxy.JumpHost = strings.TrimPrefix(trimmed, "ProxyJump ")
		case strings.HasPrefix(trimmed, "ProxyCommand "):
			parseProxyCommand(strings.TrimPrefix(trimmed, "ProxyCommand "), &cfg.Proxy)
		}
	}

	if !found {
		return HostConfig{}, fmt.Errorf("no sshady-managed entry with alias %q", alias)
	}

	for _, part := range strings.Fields(meta) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "type" {
			cfg.Proxy.Type = proxy.Type(kv[1])
		}
	}

	return cfg, nil
}

func parseProxyCommand(cmdline string, p *proxy.Config) {
	fields := strings.Fields(cmdline)
	for i := 0; i < len(fields); i++ {
		switch fields[i] {
		case "--proxy-type":
			if i+1 < len(fields) && p.Type == "" {
				p.Type = proxy.Type(fields[i+1])
			}
			i++
		case "--proxy":
			if i+1 < len(fields) {
				host, port := fields[i+1], ""
				if idx := strings.LastIndex(host, ":"); idx != -1 {
					host, port = host[:idx], host[idx+1:]
				}
				p.Host = host
				p.Port = port
			}
			i++
		case "--proxy-auth":
			if i+1 < len(fields) {
				userpass := strings.SplitN(fields[i+1], ":", 2)
				p.Username = userpass[0]
				if len(userpass) == 2 {
					p.Password = userpass[1]
				}
			}
			i++
		}
	}
}

func UpdateEntry(oldAlias string, cfg HostConfig) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return err
	}

	stripped, found := removeBlock(content, oldAlias)
	if !found {
		return fmt.Errorf("no sshady-managed entry with alias %q", oldAlias)
	}

	if cfg.Alias != oldAlias {
		if strings.Contains(stripped, fmt.Sprintf("# BEGIN SSHADY:%s\n", cfg.Alias)) {
			return fmt.Errorf("alias %q is already managed by sshady", cfg.Alias)
		}
		if err := checkAliasFree(stripped, cfg.Alias); err != nil {
			return err
		}
	}

	if err := backup(configPath, content); err != nil {
		return err
	}

	return atomicWrite(configPath, appendBlock(stripped, cfg.Block()), 0600)
}

func DeleteEntry(alias string) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	content, err := readConfig(configPath)
	if err != nil {
		return err
	}

	stripped, found := removeBlock(content, alias)
	if !found {
		return fmt.Errorf("no sshady-managed entry with alias %q", alias)
	}

	if err := backup(configPath, content); err != nil {
		return err
	}

	return atomicWrite(configPath, stripped, 0600)
}

func checkAliasFree(content, alias string) error {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^Host\s+%s(\s|$)`, regexp.QuoteMeta(alias)))
	if re.MatchString(content) {
		return fmt.Errorf("alias %q already exists in ~/.ssh/config (not managed by sshady)", alias)
	}
	return nil
}

func backup(configPath, content string) error {
	if content == "" {
		return nil
	}
	if err := os.WriteFile(configPath+".sshady.bak", []byte(content), 0600); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	return nil
}

func appendBlock(content, block string) string {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	return content + block
}

func removeBlock(content, alias string) (string, bool) {
	begin := "# BEGIN SSHADY:" + alias
	end := "# END SSHADY:" + alias

	var out []string
	inBlock := false
	found := false

	for _, line := range strings.Split(content, "\n") {
		switch line {
		case begin:
			inBlock = true
			found = true
			continue
		case end:
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}

	if !found {
		return content, false
	}

	result := strings.TrimRight(strings.Join(out, "\n"), "\n")
	if result != "" {
		result += "\n"
	}
	return result, true
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
	if p := os.Getenv("SSHADY_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

func ensureSSHDir() error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(configPath), 0700)
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
