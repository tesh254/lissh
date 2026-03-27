package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverKnownHosts(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsPath := filepath.Join(tmpDir, "known_hosts")

	knownHostsContent := `server1.example.com,192.168.1.1 ssh-rsa AAAAB3NzaC1...
server2.example.com ssh-rsa AAAAB3NzaC1...
[192.168.1.100]:2222 ssh-rsa AAAAB3NzaC1...`

	if err := os.WriteFile(knownHostsPath, []byte(knownHostsContent), 0600); err != nil {
		t.Fatalf("failed to write known_hosts: %v", err)
	}

	config := &DiscoveryConfig{
		SSHDir:         tmpDir,
		KnownHostsPath: knownHostsPath,
		SSHConfigPath:  filepath.Join(tmpDir, "config"),
	}

	disc := New(config)
	hosts, err := disc.discoverKnownHosts()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if len(hosts) < 1 {
		t.Fatalf("expected at least 1 host, got %d", len(hosts))
	}
}

func TestDiscoverSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	sshConfigPath := filepath.Join(tmpDir, "config")

	sshConfigContent := `Host web-server
    HostName 192.168.1.10
    Port 22
    User admin

Host db-server
    HostName 192.168.1.20
    Port 2222
    User dbadmin

Host *
    ServerAliveInterval 60
`

	if err := os.WriteFile(sshConfigPath, []byte(sshConfigContent), 0600); err != nil {
		t.Fatalf("failed to write ssh config: %v", err)
	}

	config := &DiscoveryConfig{
		SSHDir:         tmpDir,
		KnownHostsPath: filepath.Join(tmpDir, "known_hosts"),
		SSHConfigPath:  sshConfigPath,
	}

	disc := New(config)
	hosts, err := disc.discoverSSHConfig()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	if hosts[0].Hostname != "web-server" {
		t.Errorf("expected hostname web-server, got %s", hosts[0].Hostname)
	}
	if hosts[0].Port != 22 {
		t.Errorf("expected port 22, got %d", hosts[0].Port)
	}

	if hosts[1].Hostname != "db-server" {
		t.Errorf("expected hostname db-server, got %s", hosts[1].Hostname)
	}
	if hosts[1].Port != 2222 {
		t.Errorf("expected port 2222, got %d", hosts[1].Port)
	}
}

func TestDiscoverAll(t *testing.T) {
	tmpDir := t.TempDir()

	knownHostsPath := filepath.Join(tmpDir, "known_hosts")
	if err := os.WriteFile(knownHostsPath, []byte("server1.example.com ssh-rsa AAAAB3NzaC1...\n"), 0600); err != nil {
		t.Fatalf("failed to write known_hosts: %v", err)
	}

	sshConfigPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(sshConfigPath, []byte("Host web-server\n    HostName 192.168.1.10\n"), 0600); err != nil {
		t.Fatalf("failed to write ssh config: %v", err)
	}

	config := &DiscoveryConfig{
		SSHDir:         tmpDir,
		KnownHostsPath: knownHostsPath,
		SSHConfigPath:  sshConfigPath,
	}

	disc := New(config)
	hosts, err := disc.DiscoverAll()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
}

func TestParseKnownHostsLine(t *testing.T) {
	tests := []struct {
		line     string
		expected *DiscoveredHost
	}{
		{
			line: "server1.example.com ssh-rsa AAAAB3NzaC1...",
			expected: &DiscoveredHost{
				Hostname:  "server1.example.com",
				IPAddress: "",
				Port:      22,
				Source:    "known_hosts",
			},
		},
		{
			line: "192.168.1.1 ssh-rsa AAAAB3NzaC1...",
			expected: &DiscoveredHost{
				Hostname:  "192.168.1.1",
				IPAddress: "192.168.1.1",
				Port:      22,
				Source:    "known_hosts",
			},
		},
	}

	for _, tt := range tests {
		result := parseKnownHostsLine(tt.line)
		if result == nil {
			t.Fatalf("expected result, got nil for line: %s", tt.line)
		}
		if result.Hostname != tt.expected.Hostname {
			t.Errorf("expected hostname %s, got %s", tt.expected.Hostname, result.Hostname)
		}
	}
}
