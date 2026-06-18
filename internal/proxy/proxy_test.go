package proxy

import (
	"testing"
)

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"empty", "", true},
		{"localhost", "localhost", false},
		{"valid hostname", "proxy.example.com", false},
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv6", "::1", false},
		{"valid IPv6 bracketed", "[::1]", false},
		{"command injection semicolon", "evil;rm -rf /", true},
		{"command injection pipe", "evil|cat /etc/passwd", true},
		{"command injection backtick", "evil`id`", true},
		{"command injection dollar", "evil$(id)", true},
		{"spaces", "evil host", true},
		{"newline", "evil\nhost", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHost(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
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
		{"valid", "1080", false},
		{"valid 22", "22", false},
		{"empty", "", true},
		{"not numeric", "abc", true},
		{"negative", "-1", true},
		{"command injection", "80;id", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
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
		{"valid username", "alice", false},
		{"valid with underscore", "proxy_user", false},
		{"valid with dot", "admin.test", false},
		{"valid with hyphen", "service-account", false},
		{"valid with at", "user@domain", false},
		{"injection semicolon", "evil;id", true},
		{"injection pipe", "evil|id", true},
		{"injection space", "evil id", true},
		{"injection newline", "evil\nid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserPass(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserPass(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
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
		{
			name:    "valid SOCKS5",
			cfg:     Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
			wantErr: false,
		},
		{
			name:    "valid SOCKS5 with auth",
			cfg:     Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"},
			wantErr: false,
		},
		{
			name:    "valid HTTP",
			cfg:     Config{Type: TypeHTTP, Host: "proxy.example.com", Port: "8080"},
			wantErr: false,
		},
		{
			name:    "valid Tor",
			cfg:     Config{Type: TypeTor},
			wantErr: false,
		},
		{
			name:    "valid Jump",
			cfg:     Config{Type: TypeJump, JumpHost: "bastion@10.0.0.1"},
			wantErr: false,
		},
		{
			name:    "invalid type",
			cfg:     Config{Type: "invalid"},
			wantErr: true,
		},
		{
			name:    "SOCKS5 missing host",
			cfg:     Config{Type: TypeSOCKS5, Port: "1080"},
			wantErr: true,
		},
		{
			name:    "SOCKS5 invalid host",
			cfg:     Config{Type: TypeSOCKS5, Host: "evil;id", Port: "1080"},
			wantErr: true,
		},
		{
			name:    "Jump missing host",
			cfg:     Config{Type: TypeJump},
			wantErr: true,
		},
		{
			name:    "SOCKS5 invalid port",
			cfg:     Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "abc"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
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
			name: "SOCKS5 no auth",
			cfg:  Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
			want: "ncat --proxy-type socks5 --proxy proxy.example.com:1080 %h %p",
		},
		{
			name: "SOCKS5 with auth",
			cfg:  Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice", Password: "s3cr3t"},
			want: "ncat --proxy-type socks5 --proxy proxy.example.com:1080 --proxy-auth alice:s3cr3t %h %p",
		},
		{
			name: "HTTP no auth",
			cfg:  Config{Type: TypeHTTP, Host: "proxy.example.com", Port: "8080"},
			want: "ncat --proxy-type http --proxy proxy.example.com:8080 %h %p",
		},
		{
			name: "Tor",
			cfg:  Config{Type: TypeTor},
			want: "ncat --proxy-type socks5 --proxy 127.0.0.1:9050 %h %p",
		},
		{
			name: "Jump returns empty",
			cfg:  Config{Type: TypeJump, JumpHost: "bastion"},
			want: "",
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
	cfg := Config{Type: TypeSOCKS5, Host: "proxy.example.com", Port: "1080", Username: "alice"}
	got := cfg.Summary()
	want := "SOCKS5 via proxy.example.com:1080 (authenticated)"
	if got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}
