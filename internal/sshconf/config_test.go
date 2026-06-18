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
			"fully valid",
			HostConfig{
				Alias: "myserver", HostName: "10.0.0.1", User: "admin", Port: "22",
				Proxy: proxy.Config{Type: proxy.TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
			},
			false,
		},
		{"missing alias", HostConfig{HostName: "10.0.0.1", User: "admin", Port: "22", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
		{"bad alias space", HostConfig{Alias: "my server", HostName: "10.0.0.1", User: "admin", Port: "22", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
		{"bad alias newline", HostConfig{Alias: "foo\nHost evil", HostName: "10.0.0.1", User: "admin", Port: "22", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
		{"missing hostname", HostConfig{Alias: "myserver", User: "admin", Port: "22", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
		{"bad hostname", HostConfig{Alias: "myserver", HostName: "evil;rm -rf /", User: "admin", Port: "22", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
		{"bad user", HostConfig{Alias: "myserver", HostName: "10.0.0.1", User: "evil;id", Port: "22", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
		{"bad port", HostConfig{Alias: "myserver", HostName: "10.0.0.1", User: "admin", Port: "abc", Proxy: proxy.Config{Type: proxy.TypeTor}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostConfig() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock(t *testing.T) {
	cfg := HostConfig{
		Alias: "myserver", HostName: "10.0.0.1", User: "admin", Port: "22",
		Proxy: proxy.Config{Type: proxy.TypeSOCKS5, Host: "proxy.example.com", Port: "1080"},
	}
	block := cfg.Block()
	if block == "" {
		t.Fatal("Block() returned empty")
	}
	for _, want := range []string{
		"# BEGIN SSHADY:myserver",
		"# END SSHADY:myserver",
		"Host myserver",
		"HostName 10.0.0.1",
		"User admin",
		"Port 22",
		"ProxyCommand ncat",
	} {
		if !contains(block, want) {
			t.Errorf("Block() missing %q", want)
		}
	}
}

func TestBlockJump(t *testing.T) {
	cfg := HostConfig{
		Alias: "internal", HostName: "10.0.0.5", User: "admin", Port: "22",
		Proxy: proxy.Config{Type: proxy.TypeJump, JumpHost: "bastion@192.168.1.1"},
	}
	block := cfg.Block()
	if !contains(block, "ProxyJump bastion@192.168.1.1") {
		t.Errorf("Block() missing ProxyJump, got:\n%s", block)
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config")
	content := "Host test\n    HostName 1.2.3.4\n"

	if err := atomicWrite(path, content, 0644); err != nil {
		t.Fatalf("atomicWrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != content {
		t.Errorf("got %q, want %q", string(data), content)
	}

	// No temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if contains(e.Name(), ".sshady-tmp-") {
			t.Errorf("temp file leaked: %s", e.Name())
		}
	}
}

func TestWriteAndRemoveEntry(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	SetConfigPath(configPath)
	defer SetConfigPath("")

	cfg := HostConfig{
		Alias: "testserver", HostName: "10.0.0.100", User: "admin", Port: "22",
		Proxy: proxy.Config{Type: proxy.TypeTor},
	}

	if err := WriteEntry(cfg, false); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	// Verify it was written
	data, _ := os.ReadFile(configPath)
	if !contains(string(data), "# BEGIN SSHADY:testserver") {
		t.Fatal("entry not written")
	}

	// Verify backup exists
	bakDir, _ := os.ReadDir(dir)
	hasBackup := false
	for _, e := range bakDir {
		if contains(e.Name(), ".sshady.") && contains(e.Name(), ".bak") {
			hasBackup = true
			break
		}
	}
	if !hasBackup {
		t.Error("no backup file created")
	}

	// Remove it
	if err := RemoveEntry("testserver"); err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}

	data, _ = os.ReadFile(configPath)
	if contains(string(data), "testserver") {
		t.Error("entry still present after removal")
	}
}

func TestWriteEntryForce(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	SetConfigPath(configPath)
	defer SetConfigPath("")

	cfg := HostConfig{
		Alias: "srv", HostName: "1.1.1.1", User: "admin", Port: "22",
		Proxy: proxy.Config{Type: proxy.TypeTor},
	}

	// First write
	if err := WriteEntry(cfg, false); err != nil {
		t.Fatalf("first WriteEntry: %v", err)
	}

	// Second write without force should fail
	if err := WriteEntry(cfg, false); err == nil {
		t.Error("expected error on duplicate without --force, got nil")
	}

	// Second write with force should succeed
	if err := WriteEntry(cfg, true); err != nil {
		t.Errorf("WriteEntry with force: %v", err)
	}
}

func TestReadManagedEntries_Empty(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	SetConfigPath(configPath)
	defer SetConfigPath("")

	os.WriteFile(configPath, []byte(""), 0644)

	entries, err := ReadManagedEntries()
	if err != nil {
		t.Fatalf("ReadManagedEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestFindEntry(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	SetConfigPath(configPath)
	defer SetConfigPath("")

	cfg := HostConfig{
		Alias: "findme", HostName: "10.0.0.99", User: "test", Port: "2222",
		Proxy: proxy.Config{Type: proxy.TypeTor},
	}
	WriteEntry(cfg, false)

	entry, err := FindEntry("findme")
	if err != nil {
		t.Fatalf("FindEntry: %v", err)
	}
	if entry == nil {
		t.Fatal("FindEntry returned nil")
	}
	if entry.Port != "2222" {
		t.Errorf("expected port 2222, got %q", entry.Port)
	}

	// Non-existent
	entry, err = FindEntry("nonexistent")
	if err != nil {
		t.Fatalf("FindEntry: %v", err)
	}
	if entry != nil {
		t.Error("expected nil for non-existent entry")
	}
}

func TestBackupRotation(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	SetConfigPath(configPath)
	defer SetConfigPath("")

	// Create 7 backups
	for i := 0; i < 7; i++ {
		cfg := HostConfig{
			Alias: "srv", HostName: "1.1.1.1", User: "admin", Port: "22",
			Proxy: proxy.Config{Type: proxy.TypeTor},
		}
		if err := WriteEntry(cfg, true); err != nil {
			t.Fatalf("WriteEntry %d: %v", i, err)
		}
	}

	// Should have at most 5 backups + current config
	entries, _ := os.ReadDir(dir)
	bakCount := 0
	for _, e := range entries {
		if contains(e.Name(), ".sshady.") && contains(e.Name(), ".bak") {
			bakCount++
		}
	}
	if bakCount > 5 {
		t.Errorf("expected <= 5 backups, got %d", bakCount)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example tests for sshconf package.

func ExampleHostConfig_Block() {
	cfg := HostConfig{
		Alias: "example", HostName: "10.0.0.1", User: "admin", Port: "22",
		Proxy: proxy.Config{Type: proxy.TypeTor},
	}
	block := cfg.Block()
	// Verify key elements are present
	fmt.Println(contains(block, "# BEGIN SSHADY:example"))
	fmt.Println(contains(block, "Host example"))
	fmt.Println(contains(block, "ProxyCommand ncat"))
	// Output:
	// true
	// true
	// true
}
