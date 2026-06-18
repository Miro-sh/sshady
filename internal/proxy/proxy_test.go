package proxy

import (
	"fmt"
	"testing"
)

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		// Valid cases
		{"empty", "", true},
		{"localhost", "localhost", false},
		{"simple hostname", "proxy", false},
		{"fqdn", "proxy.example.com", false},
		{"long fqdn", "very.long.hostname.with.many.parts.example.com", false},
		{"hostname with hyphen", "my-proxy.example.com", false},
		{"IPv4", "192.168.1.1", false},
		{"IPv4 localhost", "127.0.0.1", false},
		{"IPv6 loopback", "::1", false},
		{"IPv6 full", "2001:db8::1", false},
		{"IPv6 bracketed", "[::1]", false},
		{"IPv6 bracketed full", "[2001:db8::1]", false},
		{"IPv4-mapped IPv6", "::ffff:192.0.2.1", false},

		// Injection attempts
		{"semicolon injection", "evil;rm -rf /", true},
		{"pipe injection", "evil|cat /etc/passwd", true},
		{"backtick injection", "evil`id`", true},
		{"dollar paren injection", "evil$(id)", true},
		{"newline injection", "evil\nhost", true},
		{"space injection", "evil host", true},
		{"ampersand injection", "evil&id", true},
		{"redirect injection", "evil>/tmp/pwned", true},
		{"null byte", "evil\x00host", true},
		{"tab injection", "evil\thost", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHost(%q) error = %v, wantErr = %v", tt.host, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"valid 1080", "1080", false},
		{"valid 22", "22", false},
		{"valid 1", "1", false},
		{"valid 65535", "65535", false},
		{"valid 80", "80", false},
		{"valid 443", "443", false},
		{"empty", "", true},
		{"not numeric", "abc", true},
		{"negative", "-1", true},
		{"zero", "0", true},
		{"too high", "65536", true},
		{"way too high", "99999", true},
		{"float", "80.0", true},
		{"hex", "0x50", true},
		{"injection semicolon", "80;id", true},
		{"injection space", "80 id", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%q) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserPass(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", false},
		{"simple", "alice", false},
		{"with underscore", "proxy_user", false},
		{"with dot", "admin.test", false},
		{"with hyphen", "service-account", false},
		{"with at", "user@domain", false},

		{"semicolon", "evil;id", true},
		{"pipe", "evil|id", true},
		{"space", "evil id", true},
		{"newline", "evil\nid", true},
		{"backtick", "evil`id`", true},
		{"dollar", "evil$(id)", true},
		{"slash", "evil/id", true},
		{"backslash", "evil\\id", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserPass(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserPass(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{"valid", "myserver", false},
		{"valid with dot", "my.server", false},
		{"valid with underscore", "my_server", false},
		{"valid with hyphen", "my-server", false},
		{"starts with number", "2server", false},

		{"empty", "", true},
		{"space", "my server", true},
		{"newline", "my\nserver", true},
		{"semicolon", "my;server", true},
		{"starts with dot", ".myserver", true},
		{"starts with hyphen", "-myserver", true},
		{"starts with underscore", "_myserver", true},
		{"shell injection", "x;id", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlias(tt.alias)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlias(%q) error = %v, wantErr = %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid SOCKS5", Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080"}, false},
		{"valid SOCKS5 auth", Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"}, false},
		{"valid HTTP", Config{Type: TypeHTTP, Host: "proxy.example.com", Port: "8080"}, false},
		{"valid Tor", Config{Type: TypeTor}, false},
		{"valid Jump", Config{Type: TypeJump}, false},
		{"valid SOCKS4", Config{Type: TypeSOCKS4, Host: "proxy.example.com", Port: "1080"}, false},
		{"valid SOCKS4a", Config{Type: TypeSOCKS4a, Host: "proxy.example.com", Port: "1080"}, false},
		{"SOCKS4 with auth", Config{Type: TypeSOCKS4, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"}, false},
		{"SOCKS4 default port", Config{Type: TypeSOCKS4, Host: "proxy.example.com"}, false},
		{"invalid type", Config{Type: "invalid"}, true},
		{"SOCKS5 missing host", Config{Type: TypeSOCKS5, Port: "1080"}, true},
		{"SOCKS5 bad host", Config{Type: TypeSOCKS5, Host: "evil;id", Port: "1080"}, true},
		{"SOCKS5 bad port", Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "abc"}, true},
		{"SOCKS5 port 0", Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "0"}, true},
		{"SOCKS5 port 65536", Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "65536"}, true},
		{"SOCKS4 missing host", Config{Type: TypeSOCKS4, Port: "1080"}, true},
		{"SOCKS4a missing host", Config{Type: TypeSOCKS4a, Port: "1080"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestProxyCommand(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			"SOCKS5 no auth",
			Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
			"ncat --proxy-type socks5 --proxy proxy.example.com:1080 %h %p",
		},
		{
			"SOCKS5 with auth",
			Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"},
			"ncat --proxy-type socks5 --proxy proxy.example.com:1080 --proxy-auth alice:s3cr3t %h %p",
		},
		{
			"HTTP",
			Config{Type: TypeHTTP, Host: "proxy.example.com", Port: "8080"},
			"ncat --proxy-type http --proxy proxy.example.com:8080 %h %p",
		},
		{
			"Tor",
			Config{Type: TypeTor},
			"ncat --proxy-type socks5 --proxy 127.0.0.1:9050 %h %p",
		},
		{
			"SOCKS4 no auth",
			Config{Type: TypeSOCKS4, Host: "proxy.example.com", Port: "1080"},
			"ncat --proxy-type socks4 --proxy proxy.example.com:1080 %h %p",
		},
		{
			"SOCKS4 with auth ignored",
			Config{Type: TypeSOCKS4, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"},
			"ncat --proxy-type socks4 --proxy proxy.example.com:1080 %h %p",
		},
		{
			"SOCKS4a",
			Config{Type: TypeSOCKS4a, Host: "proxy.example.com", Port: "1080"},
			"ncat --proxy-type socks4 --proxy proxy.example.com:1080 %h %p",
		},
		{
			"Jump returns empty",
			Config{Type: TypeJump},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ProxyCommand()
			if got != tt.want {
				t.Errorf("ProxyCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"SOCKS5 with port in host", Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080"}, "SOCKS5 via proxy.example.com:1080"},
		{"SOCKS5 auth", Config{Type: TypeSOCKS5, Host: "p:1080", Username: "a"}, "SOCKS5 via p:1080 (authenticated)"},
		{"HTTP", Config{Type: TypeHTTP, Host: "p:8080"}, "HTTP CONNECT via p:8080"},
		{"Tor", Config{Type: TypeTor}, "Tor via 127.0.0.1:9050"},
		{"Jump", Config{Type: TypeJump}, "SSH jump host"},
		{"SOCKS4", Config{Type: TypeSOCKS4, Host: "proxy.example.com", Port: "1080"}, "SOCKS4 via proxy.example.com:1080"},
		{"SOCKS4a", Config{Type: TypeSOCKS4a, Host: "proxy.example.com", Port: "1080"}, "SOCKS4a via proxy.example.com:1080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.Summary()
			if got != tt.want {
				t.Errorf("Summary() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Fuzz tests
func FuzzValidateHost(f *testing.F) {
	seeds := []string{"", "localhost", "example.com", "192.168.1.1", "::1", "evil;id", "x\nx"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic
		_ = ValidateHost(s)
	})
}

func FuzzValidatePort(f *testing.F) {
	seeds := []string{"", "0", "22", "1080", "65535", "65536", "abc", "80;id"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = ValidatePort(s)
	})
}

func FuzzValidateUserPass(f *testing.F) {
	seeds := []string{"", "alice", "evil;id", "x\nx", "x y"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = ValidateUserPass(s)
	})
}

// Benchmarks
func BenchmarkValidateHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateHost("proxy.example.com")
	}
}

func BenchmarkValidatePort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidatePort("1080")
	}
}

func BenchmarkProxyCommand(b *testing.B) {
	cfg := Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"}
	for i := 0; i < b.N; i++ {
		_ = cfg.ProxyCommand()
	}
}

// Example tests — serve as documentation and are verified by 'go test'.

func ExampleValidateHost() {
	fmt.Println(ValidateHost("example.com"))
	fmt.Println(ValidateHost("evil;rm -rf /"))
	// Output:
	// <nil>
	// invalid host "evil;rm -rf /": must be a valid hostname, IPv4, or IPv6 address
}

func ExampleValidatePort() {
	fmt.Println(ValidatePort("1080"))
	fmt.Println(ValidatePort("0"))
	// Output:
	// <nil>
	// invalid port "0": must be between 1 and 65535
}

func ExampleConfig_ProxyCommand() {
	cfg := Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080"}
	fmt.Println(cfg.ProxyCommand())
	// Output: ncat --proxy-type socks5 --proxy proxy.example.com:1080 %h %p
}

func ExampleConfig_Summary() {
	cfg := Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice"}
	fmt.Println(cfg.Summary())
	// Output: SOCKS5 via proxy.example.com:1080 (authenticated)
}
