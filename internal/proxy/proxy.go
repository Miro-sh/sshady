package proxy

import (
	"fmt"
	"regexp"
	"strings"
)

// Type represents a proxy type.
type Type string

const (
	TypeSOCKS5 Type = "socks5"
	TypeHTTP   Type = "http"
	TypeTor    Type = "tor"
	TypeJump   Type = "jump"
)

// AllowedTypes is the set of valid proxy types.
var AllowedTypes = map[Type]bool{
	TypeSOCKS5: true,
	TypeHTTP:   true,
	TypeTor:    true,
	TypeJump:   true,
}

// Validation regexes to prevent command injection in ncat proxy commands.
var (
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	ipv4Regex     = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)
	ipv6Regex     = regexp.MustCompile(`^\[?[0-9a-fA-F:.]+\]?$`)
	portRegex     = regexp.MustCompile(`^\d{1,5}$`)
	userPassRegex = regexp.MustCompile(`^[a-zA-Z0-9_.@-]+$`)
)

// ValidateHost checks a host string is safe (hostname, IPv4, or IPv6).
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if host == "localhost" {
		return nil
	}
	if hostnameRegex.MatchString(host) {
		return nil
	}
	if ipv4Regex.MatchString(host) {
		return nil
	}
	if ipv6Regex.MatchString(host) {
		return nil
	}
	return fmt.Errorf("invalid host %q: must be a valid hostname, IPv4, or IPv6 address", host)
}

// ValidatePort checks a port string is a valid integer 1-65535.
func ValidatePort(port string) error {
	if !portRegex.MatchString(port) {
		return fmt.Errorf("invalid port %q: must be numeric", port)
	}
	return nil
}

// ValidateUserPass checks a username or password is safe for ncat command line.
func ValidateUserPass(s string) error {
	if s == "" {
		return nil
	}
	if userPassRegex.MatchString(s) {
		return nil
	}
	return fmt.Errorf("invalid characters in credential: only alphanumeric, underscore, dot, hyphen, and @ are allowed")
}

// Config holds proxy configuration for a single SSH host.
type Config struct {
	Type     Type
	Host     string
	Port     string
	Username string
	Password string
	JumpHost string
}

// Validate checks the Config for security and correctness.
func (c Config) Validate() error {
	if !AllowedTypes[c.Type] {
		return fmt.Errorf("unknown proxy type %q", c.Type)
	}

	switch c.Type {
	case TypeSOCKS5, TypeHTTP:
		if err := ValidateHost(c.Host); err != nil {
			return fmt.Errorf("proxy host: %w", err)
		}
		if err := ValidatePort(c.Port); err != nil {
			return fmt.Errorf("proxy port: %w", err)
		}
		if err := ValidateUserPass(c.Username); err != nil {
			return fmt.Errorf("proxy username: %w", err)
		}
		if err := ValidateUserPass(c.Password); err != nil {
			return fmt.Errorf("proxy password: %w", err)
		}
	case TypeJump:
		if c.JumpHost == "" {
			return fmt.Errorf("jump host is required for proxy type 'jump'")
		}
		parts := strings.SplitN(c.JumpHost, "@", 2)
		if len(parts) == 2 {
			if err := ValidateUserPass(parts[0]); err != nil {
				return fmt.Errorf("jump host username: %w", err)
			}
			if err := ValidateHost(parts[1]); err != nil {
				return fmt.Errorf("jump host hostname: %w", err)
			}
		} else {
			if err := ValidateHost(c.JumpHost); err != nil {
				return fmt.Errorf("jump host: %w", err)
			}
		}
	case TypeTor:
		// Tor uses hardcoded 127.0.0.1:9050, no external input
	}
	return nil
}

// ProxyCommand returns the ProxyCommand directive for use in ~/.ssh/config.
// Returns an empty string for TypeJump (use ProxyJump instead).
func (c Config) ProxyCommand() string {
	switch c.Type {
	case TypeSOCKS5:
		base := fmt.Sprintf("ncat --proxy-type socks5 --proxy %s:%s", c.Host, c.Port)
		if c.Username != "" {
			base += fmt.Sprintf(" --proxy-auth %s:%s", c.Username, c.Password)
		}
		return base + " %h %p"
	case TypeHTTP:
		base := fmt.Sprintf("ncat --proxy-type http --proxy %s:%s", c.Host, c.Port)
		if c.Username != "" {
			base += fmt.Sprintf(" --proxy-auth %s:%s", c.Username, c.Password)
		}
		return base + " %h %p"
	case TypeTor:
		return "ncat --proxy-type socks5 --proxy 127.0.0.1:9050 %h %p"
	}
	return ""
}

// Summary returns a human-readable summary of the proxy configuration.
func (c Config) Summary() string {
	switch c.Type {
	case TypeSOCKS5:
		s := fmt.Sprintf("SOCKS5 via %s:%s", c.Host, c.Port)
		if c.Username != "" {
			s += " (authenticated)"
		}
		return s
	case TypeHTTP:
		s := fmt.Sprintf("HTTP CONNECT via %s:%s", c.Host, c.Port)
		if c.Username != "" {
			s += " (authenticated)"
		}
		return s
	case TypeTor:
		return "Tor (127.0.0.1:9050)"
	case TypeJump:
		return fmt.Sprintf("SSH Jump -> %s", c.JumpHost)
	}
	return "none"
}

// MetaString returns metadata for the SSHADY META comment in ~/.ssh/config.
func (c Config) MetaString() string {
	switch c.Type {
	case TypeSOCKS5:
		auth := "no"
		if c.Username != "" {
			auth = "yes"
		}
		return fmt.Sprintf("type=socks5 addr=%s:%s auth=%s", c.Host, c.Port, auth)
	case TypeHTTP:
		auth := "no"
		if c.Username != "" {
			auth = "yes"
		}
		return fmt.Sprintf("type=http addr=%s:%s auth=%s", c.Host, c.Port, auth)
	case TypeTor:
		return "type=tor addr=127.0.0.1:9050 auth=no"
	case TypeJump:
		return fmt.Sprintf("type=jump host=%s", c.JumpHost)
	}
	return ""
}
