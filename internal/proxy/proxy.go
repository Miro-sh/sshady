package proxy

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// ── Proxy type enum ──────────────────────────────────────────

// Type represents a supported proxy type.
type Type string

const (
	TypeSOCKS5 Type = "socks5"
	TypeHTTP   Type = "http"
	TypeTor    Type = "tor"
	TypeJump   Type = "jump"
)

// AllowedTypes maps proxy type strings to their Type values.
var AllowedTypes = map[string]Type{
	"jump":   TypeJump,
	"socks5": TypeSOCKS5,
	"http":   TypeHTTP,
	"tor":    TypeTor,
}

// NcatProxyType maps internal proxy types to ncat --proxy-type values.
var NcatProxyType = map[Type]string{
	TypeSOCKS5: "socks5",
	TypeHTTP:   "http-connect",
}

// ── Validation regexes ────────────────────────────────────────

var (
	hostnameRegex  = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	aliasRegex     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,252}$`)
	userPassRegex  = regexp.MustCompile(`^[a-zA-Z0-9_.@-]+$`)
)

// ── Structured error types ────────────────────────────────────

// ValidationError represents an input validation failure.
type ValidationError struct {
	Field  string // Which field failed validation
	Value  string // The invalid value (may be truncated)
	Reason string // Human-readable reason
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s %q: %s", e.Field, e.Value, e.Reason)
}

// ErrInvalidHost returns a ValidationError for an invalid hostname/IP.
func ErrInvalidHost(host string) error {
	return &ValidationError{Field: "host", Value: host, Reason: "must be a valid hostname, IPv4, or IPv6 address"}
}

// ErrInvalidPort returns a ValidationError for an invalid port.
func ErrInvalidPort(port string) error {
	return &ValidationError{Field: "port", Value: port, Reason: "must be between 1 and 65535"}
}

// ErrInvalidAlias returns a ValidationError for an invalid alias.
func ErrInvalidAlias(alias string) error {
	return &ValidationError{Field: "alias", Value: alias, Reason: "must start with a letter or digit, contain only [a-zA-Z0-9._-], max 253 chars"}
}

// ErrInvalidCredential returns a ValidationError for an invalid username or password.
func ErrInvalidCredential(field, value string) error {
	return &ValidationError{Field: field, Value: value, Reason: "must contain only [a-zA-Z0-9_.@-]"}
}

// ── Config type ───────────────────────────────────────────────

// Config holds all configuration for a proxy.
type Config struct {
	Type     Type   // socks5, http, tor, jump
	Host     string // Proxy hostname or IP (empty for tor/jump)
	Port     string // Proxy port (default: 1080 for socks5/http, 9050 for tor)
	Username string // Proxy auth username
	Password string // Proxy auth password
}

// ── Validation ────────────────────────────────────────────────

// ValidateHost validates a hostname or IP address.
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if host == "localhost" {
		return nil
	}
	// Strip brackets from IPv6 like [::1]
	stripped := host
	if len(stripped) > 2 && stripped[0] == '[' && stripped[len(stripped)-1] == ']' {
		stripped = stripped[1 : len(stripped)-1]
	}
	if ip := net.ParseIP(stripped); ip != nil {
		return nil
	}
	if hostnameRegex.MatchString(host) {
		return nil
	}
	return ErrInvalidHost(host)
}

// ValidatePort validates a port number string.
func ValidatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return ErrInvalidPort(port)
	}
	if n < 1 || n > 65535 {
		return ErrInvalidPort(port)
	}
	return nil
}

// ValidateAlias validates an SSH host alias.
func ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if !aliasRegex.MatchString(alias) {
		return ErrInvalidAlias(alias)
	}
	return nil
}

// ValidateUserPass validates a username or password for shell safety.
func ValidateUserPass(value, field string) error {
	if value == "" {
		return nil // credentials are optional
	}
	if !userPassRegex.MatchString(value) {
		return ErrInvalidCredential(field, value)
	}
	return nil
}

// Validate checks all fields of the Config and applies defaults.
// Uses pointer receiver so defaults are visible to the caller.
func (c *Config) Validate() error {
	// Apply defaults
	if c.Host == "" {
		if c.Type == TypeTor {
			c.Host = "127.0.0.1"
		}
	}
	if c.Port == "" {
		switch c.Type {
		case TypeSOCKS5, TypeHTTP:
			c.Port = "1080"
		case TypeTor:
			c.Port = "9050"
		}
	}

	switch c.Type {
	case TypeSOCKS5, TypeHTTP, TypeTor:
		if err := ValidateHost(c.Host); err != nil {
			return err
		}
		if err := ValidatePort(c.Port); err != nil {
			return err
		}
	case TypeJump:
		// Jump host validation happens at the HostConfig level
		return nil
	default:
		return fmt.Errorf("unknown proxy type %q", c.Type)
	}

	if c.Username != "" {
		if err := ValidateUserPass(c.Username, "username"); err != nil {
			return err
		}
	}
	if c.Password != "" {
		if err := ValidateUserPass(c.Password, "password"); err != nil {
			return err
		}
	}
	return nil
}

// ── Command generation ────────────────────────────────────────

// ProxyCommand generates the ncat ProxyCommand string for SSH config.
func (c Config) ProxyCommand() string {
	if c.Type == TypeJump {
		return "" // jump hosts use ProxyJump, not ProxyCommand
	}
	if c.Type == TypeTor {
		host := c.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := c.Port
		if port == "" {
			port = "9050"
		}
		return fmt.Sprintf("ncat --proxy-type socks5 --proxy %s:%s %%h %%p", host, port)
	}
	host := c.Host
	port := c.Port
	if port == "" {
		port = "1080"
	}
	base := fmt.Sprintf("ncat --proxy-type %s --proxy %s:%s",
		NcatProxyType[c.Type], host, port)
	if c.Username != "" {
		base += fmt.Sprintf(" --proxy-auth %s", c.Username)
		if c.Password != "" {
			base += fmt.Sprintf(":%s", c.Password)
		}
	}
	base += " %h %p"
	return base
}

// Summary returns a human-readable one-line description of the proxy config.
func (c Config) Summary() string {
	host := c.Host
	port := c.Port
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		if c.Type == TypeTor {
			port = "9050"
		} else {
			port = "1080"
		}
	}
	addr := host + ":" + port
	auth := ""
	if c.Username != "" {
		auth = " (authenticated)"
	}
	switch c.Type {
	case TypeSOCKS5:
		return "SOCKS5 via " + addr + auth
	case TypeHTTP:
		return "HTTP CONNECT via " + addr + auth
	case TypeTor:
		return "Tor via " + addr
	case TypeJump:
		return "SSH jump host"
	default:
		return string(c.Type) + " via " + addr
	}
}

// MetaString generates the SSHADY META comment line for the config block.
func (c Config) MetaString() string {
	auth := "no"
	if c.Username != "" {
		auth = "yes"
	}
	return fmt.Sprintf("# SSHADY META: type=%s addr=%s:%s auth=%s",
		c.Type, c.Host, c.Port, auth)
}
