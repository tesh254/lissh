package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/pkg/style"
)

var rootCmd *cobra.Command

func NewCompletionCmd(cmd *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Install shell autocompletion scripts",
		Long: `Install shell autocompletion scripts for lissh.

Automatically detects your shell and installs the completion script.

Examples:
  # Auto-detect shell and install
  lissh completion install

  # Install for specific shell
  lissh completion install zsh
  lissh completion install bash
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	bashCmd := &cobra.Command{
		Use:   "bash",
		Short: "Install bash completion",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return installForShell("bash") },
	}

	zshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "Install zsh completion",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return installForShell("zsh") },
	}

	fishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Install fish completion",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return installForShell("fish") },
	}

	powershellCmd := &cobra.Command{
		Use:   "powershell",
		Short: "Install powershell completion",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return installForShell("powershell") },
	}

	installCmd := &cobra.Command{
		Use:   "install [shell]",
		Short: "Automatically install completion for your shell",
		Long: `Automatically detect your shell and install completions.

If shell is not specified, detects from SHELL environment variable.

Supported shells: bash, zsh, fish, powershell
`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInstall,
	}

	completionCmd.AddCommand(bashCmd, zshCmd, fishCmd, powershellCmd, installCmd)

	return completionCmd
}

func SetRootCmd(cmd *cobra.Command) {
	rootCmd = cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	shell := detectShell()
	if len(args) > 0 {
		shell = args[0]
	}

	shell = strings.ToLower(shell)

	return installForShell(shell)
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		parts := strings.Split(shell, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return "bash"
}

func installForShell(shell string) error {
	var completionPath string
	var err error

	switch shell {
	case "bash":
		completionPath, err = installBashCompletion()
	case "zsh":
		completionPath, err = installZshCompletion()
	case "fish":
		completionPath, err = installFishCompletion()
	case "powershell", "pwsh":
		completionPath, err = installPowerShellCompletion()
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shell)
	}

	if err != nil {
		return err
	}

	fmt.Printf("  %s Installed %s completion to %s\n", style.OK.Render("✓"), shell, completionPath)
	return nil
}

func installBashCompletion() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	completionDir := filepath.Join(home, ".bash_completion.d")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	completionPath := filepath.Join(completionDir, "lissh")

	var buf strings.Builder
	if err := rootCmd.GenBashCompletion(&buf); err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	if err := os.WriteFile(completionPath, []byte(buf.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Printf("  %s Add this to your ~/.bashrc:\n", style.Info.Render("→"))
	fmt.Printf("     source %s\n", completionPath)

	return completionPath, nil
}

func installZshCompletion() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	zshDir := filepath.Join(home, ".zshrc.d")
	if err := os.MkdirAll(zshDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	completionDir := filepath.Join(zshDir, "completions")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	completionPath := filepath.Join(completionDir, "_lissh")

	var buf strings.Builder
	if err := rootCmd.GenZshCompletion(&buf); err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	if err := os.WriteFile(completionPath, []byte(buf.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Printf("  %s Add this to your ~/.zshrc:\n", style.Info.Render("→"))
	fmt.Printf("     autoload -U compinit && compinit\n")
	fmt.Printf("     source %s\n", filepath.Join(zshDir, "_lissh"))
	fmt.Printf("     fpath+=( %s )\n", completionDir)

	return completionPath, nil
}

func installFishCompletion() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	var configHome string
	if runtime.GOOS == "darwin" {
		configHome = filepath.Join(home, ".config", "fish")
	} else {
		configHome = os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config", "fish")
		}
	}

	completionDir := filepath.Join(configHome, "completions")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	completionPath := filepath.Join(completionDir, "lissh.fish")

	var buf strings.Builder
	if err := rootCmd.GenFishCompletion(&buf, true); err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	if err := os.WriteFile(completionPath, []byte(buf.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Printf("  %s Fish completions are automatically loaded on restart.\n", style.Info.Render("→"))

	return completionPath, nil
}

func installPowerShellCompletion() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	completionDir := filepath.Join(home, "Documents", "PowerShell", "Complete")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	completionPath := filepath.Join(completionDir, "lissh.ps1")

	var buf strings.Builder
	if err := rootCmd.GenPowerShellCompletion(&buf); err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	if err := os.WriteFile(completionPath, []byte(buf.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Printf("  %s Add this to your PowerShell profile:\n", style.Info.Render("→"))
	fmt.Printf("     . %s\n", completionPath)

	return completionPath, nil
}
