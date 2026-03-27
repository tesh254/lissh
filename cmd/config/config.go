package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/sshconfig"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "SSH config helpers",
		Long:  `Manage SSH configuration with ease.`,
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current SSH config",
		RunE:  showConfig,
	}

	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Show changes lissh would make",
		RunE:  showDiff,
	}

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply recommended SSH settings",
		RunE:  applyConfig,
	}
	applyCmd.Flags().Bool("dry-run", false, "Show changes without applying")
	applyCmd.Flags().Bool("backup", true, "Create backup before applying")

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup SSH config",
		RunE:  backupConfig,
	}

	restoreCmd := &cobra.Command{
		Use:   "restore [backup-file]",
		Short: "Restore SSH config from backup",
		Args:  cobra.ExactArgs(1),
		RunE:  restoreConfig,
	}

	keepaliveCmd := &cobra.Command{
		Use:   "keepalive [on|off]",
		Short: "Enable or disable SSH keepalive",
		Args:  cobra.ExactArgs(1),
		RunE:  setKeepalive,
	}

	compressionCmd := &cobra.Command{
		Use:   "compression [on|off]",
		Short: "Enable or disable compression",
		Args:  cobra.ExactArgs(1),
		RunE:  setCompression,
	}

	controlCmd := &cobra.Command{
		Use:   "controlmaster [auto|no]",
		Short: "Set ControlMaster setting",
		Args:  cobra.ExactArgs(1),
		RunE:  setControlMaster,
	}

	cmd.AddCommand(showCmd, diffCmd, applyCmd, backupCmd, restoreCmd, keepaliveCmd, compressionCmd, controlCmd)

	return cmd
}

func showConfig(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No SSH config file found")
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	fmt.Println(string(content))
	return nil
}

func showDiff(cmd *cobra.Command, args []string) error {
	cfg, err := sshconfig.ReadSSHConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	proposed := sshconfig.RecommendedSettings()

	fmt.Println("Recommended SSH settings:")
	fmt.Println("==========================")

	for key, value := range proposed {
		current := cfg.Get(key)
		if current == "" {
			fmt.Printf("+ %s: %s (not set)\n", key, value)
		} else if current != value {
			fmt.Printf("~ %s: %s (current: %s)\n", key, value, current)
		} else {
			fmt.Printf("  %s: %s (already set)\n", key, value)
		}
	}

	return nil
}

func applyConfig(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	backup, _ := cmd.Flags().GetBool("backup")

	if dryRun {
		return showDiff(cmd, nil)
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	if backup {
		backupPath := configPath + ".lissh-backup-" + time.Now().Format("20060102-150405")
		content, err := os.ReadFile(configPath)
		if err == nil {
			os.WriteFile(backupPath, content, 0600)
			fmt.Printf("Backup created: %s\n", backupPath)
		}
	}

	proposed := sshconfig.RecommendedSettings()

	var buf bytes.Buffer
	existingContent, err := os.ReadFile(configPath)
	if err == nil {
		buf.Write(existingContent)
		if len(existingContent) > 0 && existingContent[len(existingContent)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}

	buf.WriteString("\n# Added by lissh\n")
	for key, value := range proposed {
		current := sshconfig.ExtractValue(string(existingContent), key)
		if current == "" {
			buf.WriteString(fmt.Sprintf("%s %s\n", key, value))
		}
	}

	if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("SSH config updated successfully")
	return nil
}

func backupConfig(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")
	backupDir := filepath.Join(home, ".lissh", "backups")
	backupPath := filepath.Join(backupDir, "config-"+time.Now().Format("20060102-150405"))

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No SSH config to backup")
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("Backup created: %s\n", backupPath)
	return nil
}

func restoreConfig(cmd *cobra.Command, args []string) error {
	backupPath := args[0]

	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	fmt.Printf("Config restored from: %s\n", backupPath)
	return nil
}

func setKeepalive(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lissh config keepalive [on|off]")
	}

	value := args[0]
	if value != "on" && value != "off" {
		return fmt.Errorf("value must be 'on' or 'off'")
	}

	applySetting("ServerAliveInterval", map[bool]string{true: "60", false: "0"}[value == "on"])
	applySetting("ServerAliveCountMax", map[bool]string{true: "3", false: "0"}[value == "on"])

	fmt.Printf("Keepalive %s\n", map[bool]string{true: "enabled", false: "disabled"}[value == "on"])
	return nil
}

func setCompression(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lissh config compression [on|off]")
	}

	value := args[0]
	if value != "on" && value != "off" {
		return fmt.Errorf("value must be 'on' or 'off'")
	}

	applySetting("Compression", value)

	fmt.Printf("Compression %s\n", map[bool]string{true: "enabled", false: "disabled"}[value == "on"])
	return nil
}

func setControlMaster(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lissh config controlmaster [auto|no]")
	}

	value := args[0]
	if value != "auto" && value != "no" {
		return fmt.Errorf("value must be 'auto' or 'no'")
	}

	applySetting("ControlMaster", value)
	if value == "auto" {
		home, _ := os.UserHomeDir()
		applySetting("ControlPath", filepath.Join(home, ".ssh", "sockets", "controlmaster-%r@%h-%p"))
		applySetting("ControlPersist", "10m")
	}

	fmt.Printf("ControlMaster set to %s\n", value)
	return nil
}

func applySetting(key, value string) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	content, _ := os.ReadFile(configPath)
	newContent := sshconfig.SetOrUpdateValue(string(content), key, value)

	os.WriteFile(configPath, []byte(newContent), 0600)
}

func runGitDiff(path string) error {
	cmd := exec.Command("git", "diff", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}
