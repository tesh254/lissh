package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SSHConfig struct {
	Values map[string]string
}

func ReadSSHConfig() (*SSHConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find home directory: %w", err)
	}

	configPath := filepath.Join(home, ".ssh", "config")
	return ReadSSHConfigFile(configPath)
}

func ReadSSHConfigFile(path string) (*SSHConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SSHConfig{Values: make(map[string]string)}, nil
		}
		return nil, err
	}
	defer file.Close()

	cfg := &SSHConfig{Values: make(map[string]string)}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := parts[0]
			value := strings.Join(parts[1:], " ")
			value = strings.Trim(value, "\"")
			cfg.Values[key] = value
		}
	}

	return cfg, scanner.Err()
}

func (c *SSHConfig) Get(key string) string {
	return c.Values[key]
}

func (c *SSHConfig) Has(key string) bool {
	_, exists := c.Values[key]
	return exists
}

func RecommendedSettings() map[string]string {
	return map[string]string{
		"ServerAliveInterval":   "60",
		"ServerAliveCountMax":   "3",
		"TCPKeepAlive":          "yes",
		"Compression":           "yes",
		"ControlMaster":         "auto",
		"ControlPath":           filepath.Join(os.Getenv("HOME"), ".ssh", "sockets", "controlmaster-%r@%h-%p"),
		"ControlPersist":        "10m",
		"ForwardAgent":          "no",
		"StrictHostKeyChecking": "ask",
	}
}

func ExtractValue(content, key string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.EqualFold(parts[0], key) {
			return strings.Join(parts[1:], " ")
		}
	}
	return ""
}

func SetOrUpdateValue(content, key, value string) string {
	pattern := regexp.MustCompile(`(?i)^(` + regexp.QuoteMeta(key) + `\s+)\S+.*$`)
	if pattern.MatchString(content) {
		return pattern.ReplaceAllString(content, "${1}"+value)
	}

	newLine := fmt.Sprintf("%s %s", key, value)

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += newLine + "\n"

	return content
}

func UpdateSSHConfig(key, value string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	configPath := filepath.Join(home, ".ssh", "config")
	return UpdateSSHConfigFile(configPath, key, value)
}

func UpdateSSHConfigFile(path, key, value string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			content = []byte{}
		} else {
			return err
		}
	}

	newContent := SetOrUpdateValue(string(content), key, value)

	if err := os.WriteFile(path, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func CreateBackup(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	dir := filepath.Dir(path)
	backupDir := filepath.Join(dir, ".backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupPath := filepath.Join(backupDir, filepath.Base(path)+".backup")
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

func ParseHostLine(line string) (host string, port int, err error) {
	re := regexp.MustCompile(`^\[([^\]]+)\]:?(\d+)?$`)
	matches := re.FindStringSubmatch(line)

	if matches != nil {
		host = matches[1]
		if len(matches) >= 3 && matches[2] != "" {
			fmt.Sscanf(matches[2], "%d", &port)
		} else {
			port = 22
		}
		return
	}

	parts := strings.Split(line, ":")
	host = parts[0]
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &port)
	} else {
		port = 22
	}

	return
}

func FormatHostLine(host string, port int) string {
	if port == 22 {
		return host
	}
	return fmt.Sprintf("[%s]:%d", host, port)
}
