package proxy

import "fmt"

type Type string

const (
	TypeSOCKS5 Type = "socks5"
	TypeHTTP   Type = "http"
	TypeTor    Type = "tor"
	TypeJump   Type = "jump"
)

type Config struct {
	Type     Type
	Host     string
	Port     string
	Username string
	Password string
	JumpHost string
}

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
