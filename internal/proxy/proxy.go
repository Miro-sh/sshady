// Package proxy defines proxy types, configuration, and security validation.
// All user-supplied values are validated against strict patterns to prevent
// command injection in the generated ncat ProxyCommand directives.
package proxy

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
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

// AllowedTypeStrings returns a comma-separated list for error messages.
func AllowedTypeStrings() string {
	return "socks5, http, tor, jump"
}

// ── Input validation ─────────────────────────────────────

var (
	// hostnameRegex matches RFC 1123 hostnames (labels of 1-63 alphanumeric+hyphen chars).
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	// portRegex matches a purely numeric port string.
	portRegex = regexp.MustCompile(`^\d{1,5}$`)
	// userPassRegex validates proxy/SSH credentials for ncat command-line safety.
	userPassRegex = regexp.MustCompile(`^[a-zA-Z0-9_.@-]+$`)
)

// ValidateHost checks that a host string is a safe hostname, IPv4, or IPv6 address.
// It uses Go's net package for IP parsing (authoritative) and a regex for hostnames.
// Special case: "localhost" is always allowed.
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if host == "localhost" {
		return nil
	}

	// Try parsing as IP first (net package handles all IPv4/IPv6 formats correctly)
	if ip := net.ParseIP(stripBrackets(host)); ip != nil {
		return nil
	}

	// Fall back to hostname validation
	if hostnameRegex.MatchString(host) {
		return nil
	}

	return fmt.Errorf("invalid host %q: must be a valid hostname, IPv4, or IPv6 address", host)
}

// stripBrackets removes surrounding [ ] from IPv6 addresses.
func stripBrackets(s string) string {
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}

// ValidatePort checks that a port string represents a valid TCP port (1-65535).
func ValidatePort(port string) error {
	if !portRegex.MatchString(port) {
		return fmt.Errorf("invalid port %q: must be numeric", port)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid port %q: must be between 1 and 65535", port)
	}
	return nil
}

// ValidateUserPass checks that a username or password contains only characters
// safe for ncat command-line arguments (prevents shell injection).
func ValidateUserPass(s string) error {
	if s == "" {
		return nil
	}
	if !userPassRegex.MatchString(s) {
		return fmt.Errorf("invalid characters in credential: only alphanumeric, underscore, dot, hyphen, and @ are allowed")
	}
	return nil
}

// ValidateAlias checks that an SSH Host alias is safe for ~/.ssh/config.
// Aliases can contain alphanumeric, dots, underscores, and hyphens.
var sshAliasRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-]*$`)

func ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if !sshAliasRe.MatchString(alias) {
		return fmt.Errorf("invalid alias %q: must start with alphanumeric, then alphanumeric, dots, underscores, or hyphens only", alias)
	}
	// Also reject overly long aliases
	if len(alias) > 253 {
		return fmt.Errorf("alias %q is too long (max 253 characters)", alias)
	}
	return nil
}

// ── Config ────────────────────────────────────────────────

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
// Returns a descriptive error if any field fails validation.
func (c Config) Validate() error {
	if !AllowedTypes[c.Type] {
		return fmt.Errorf("unknown proxy type %q; supported: %s", c.Type, AllowedTypeStrings())
	}

	switch c.Type {
	case TypeSOCKS5, TypeHTTP:
		if c.Host == "" {
			return fmt.Errorf("proxy host is required for type %q; use --proxy-host", c.Type)
		}
		if err := ValidateHost(c.Host); err != nil {
			return fmt.Errorf("proxy host: %w", err)
		}
		if c.Port == "" {
			c.Port = "1080"
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
			return fmt.Errorf("jump host is required for proxy type 'jump'; use --jump-host")
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
		// Tor uses hardcoded 127.0.0.1:9050, no external input to validate
	}

	return nil
}

// ── Output generators ─────────────────────────────────────

// ProxyCommand returns the ProxyCommand directive for use in ~/.ssh/config.
// The returned string is safe for SSH config because all inputs have been validated.
// Returns an empty string for TypeJump (callers should use ProxyJump instead).
func (c Config) ProxyCommand() string {
	switch c.Type {
	case TypeSOCKS5:
		return c.buildNcatCommand("socks5")
	case TypeHTTP:
		return c.buildNcatCommand("http")
	case TypeTor:
		return "ncat --proxy-type socks5 --proxy 127.0.0.1:9050 %h %p"
	}
	return ""
}

// buildNcatCommand constructs the ncat ProxyCommand for SOCKS5/HTTP proxies.
func (c Config) buildNcatCommand(proxyType string) string {
	base := fmt.Sprintf("ncat --proxy-type %s --proxy %s:%s", proxyType, c.Host, c.Port)
	if c.Username != "" {
		base += fmt.Sprintf(" --proxy-auth %s:%s", c.Username, c.Password)
	}
	return base + " %h %p"
}

// Summary returns a human-readable one-line summary of the proxy configuration.
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
