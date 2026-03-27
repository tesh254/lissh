package sshconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `Host web-server
    HostName 192.168.1.10
    Port 22
    User admin

Host *
    ServerAliveInterval 60
    Compression yes
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := ReadSSHConfigFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if cfg.Get("ServerAliveInterval") != "60" {
		t.Errorf("expected ServerAliveInterval 60, got %s", cfg.Get("ServerAliveInterval"))
	}
}

func TestRecommendedSettings(t *testing.T) {
	settings := RecommendedSettings()

	expected := map[string]string{
		"ServerAliveInterval": "60",
		"ServerAliveCountMax": "3",
		"TCPKeepAlive":        "yes",
		"Compression":         "yes",
		"ControlMaster":       "auto",
	}

	for key, value := range expected {
		if settings[key] != value {
			t.Errorf("expected %s=%s, got %s", key, value, settings[key])
		}
	}
}

func TestExtractValue(t *testing.T) {
	content := `
Host web-server
    HostName 192.168.1.10
    Port 22
    User admin
`

	if ExtractValue(content, "HostName") != "192.168.1.10" {
		t.Errorf("expected 192.168.1.10, got %s", ExtractValue(content, "HostName"))
	}

	if ExtractValue(content, "Port") != "22" {
		t.Errorf("expected 22, got %s", ExtractValue(content, "Port"))
	}

	if ExtractValue(content, "NonExistent") != "" {
		t.Errorf("expected empty string for non-existent key")
	}
}

func TestSetOrUpdateValue(t *testing.T) {
	content := `Host web-server
    HostName 192.168.1.10
    Port 22
`

	newContent := SetOrUpdateValue(content, "ServerAliveInterval", "60")

	if !contains(newContent, "ServerAliveInterval 60") {
		t.Error("expected new setting to be added")
	}

	newContent = SetOrUpdateValue(newContent, "Port", "2222")

	if !contains(newContent, "Port 2222") {
		t.Error("expected port to be updated")
	}
}

func TestUpdateSSHConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	initialContent := `Host web-server
    HostName 192.168.1.10
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	err := UpdateSSHConfigFile(configPath, "ServerAliveInterval", "60")
	if err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !contains(string(content), "ServerAliveInterval 60") {
		t.Error("expected ServerAliveInterval to be set")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
