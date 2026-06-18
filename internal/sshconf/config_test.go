package sshconf

import (
	"os"
	"path/filepath"
	"testing"

	"sshady/internal/proxy"
)

func TestValidateHostConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     HostConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: HostConfig{
				Alias:    "myserver",
				HostName: "10.0.0.1",
				User:     "admin",
				Port:     "22",
				Proxy:    proxy.Config{Type: proxy.TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
			},
			wantErr: false,
		},
		{
			name: "missing alias",
			cfg: HostConfig{
				HostName: "10.0.0.1",
				User:     "admin",
				Port:     "22",
				Proxy:    proxy.Config{Type: proxy.TypeTor},
			},
			wantErr: true,
		},
		{
			name: "invalid alias (spaces)",
			cfg: HostConfig{
				Alias:    "my server",
				HostName: "10.0.0.1",
				User:     "admin",
				Port:     "22",
				Proxy:    proxy.Config{Type: proxy.TypeTor},
			},
			wantErr: true,
		},
		{
			name: "invalid alias (injection)",
			cfg: HostConfig{
				Alias:    "foo\nHost evil",
				HostName: "10.0.0.1",
				User:     "admin",
				Port:     "22",
				Proxy:    proxy.Config{Type: proxy.TypeTor},
			},
			wantErr: true,
		},
		{
			name: "missing hostname",
			cfg: HostConfig{
				Alias: "myserver",
				User:  "admin",
				Port:  "22",
				Proxy: proxy.Config{Type: proxy.TypeTor},
			},
			wantErr: true,
		},
		{
			name: "invalid hostname",
			cfg: HostConfig{
				Alias:    "myserver",
				HostName: "evil;rm -rf /",
				User:     "admin",
				Port:     "22",
				Proxy:    proxy.Config{Type: proxy.TypeTor},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock(t *testing.T) {
	cfg := HostConfig{
		Alias:    "myserver",
		HostName: "10.0.0.1",
		User:     "admin",
		Port:     "22",
		Proxy:    proxy.Config{Type: proxy.TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
	}
	block := cfg.Block()
	if block == "" {
		t.Error("Block() returned empty string")
	}
	// Should contain markers
	if !contains(block, "# BEGIN SSHADY:myserver") {
		t.Error("Block() missing BEGIN marker")
	}
	if !contains(block, "# END SSHADY:myserver") {
		t.Error("Block() missing END marker")
	}
	if !contains(block, "Host myserver") {
		t.Error("Block() missing Host directive")
	}
	if !contains(block, "ProxyCommand ncat") {
		t.Error("Block() missing ProxyCommand")
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config")
	content := "Host test\n    HostName 1.2.3.4\n"

	err := atomicWrite(path, content, 0644)
	if err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	// Read back
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != content {
		t.Errorf("Read back = %q, want %q", string(data), content)
	}

	// Check no temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if contains(e.Name(), ".sshady-tmp-") {
			t.Errorf("Temp file left behind: %s", e.Name())
		}
	}
}

func TestReadManagedEntries_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(""), 0644)

	// Override config path for this test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	entries, err := ReadManagedEntries()
	if err != nil {
		t.Fatalf("ReadManagedEntries failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

// contains is a helper for substring check.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
