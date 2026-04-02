package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/blang/semver/v4"
	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/version"
)

var (
	autoUpdate bool
	checkOnly  bool
)

func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for or perform updates",
		Long: `Check for new releases or update lissh to the latest version.

Without flags, checks for updates. Use --install to automatically update.`,
		RunE: runUpdate,
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")
	cmd.Flags().BoolVarP(&autoUpdate, "install", "i", false, "Automatically download and install the latest version")
	cmd.Flags().BoolVar(&autoUpdate, "yes", false, "Skip confirmation prompt")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	currentVersion := version.Get()

	fmt.Printf("Current version: %s\n", version.String())

	latestVersion, err := version.CheckForUpdate()
	if err != nil {
		fmt.Printf("Failed to check for updates: %v\n", err)
		fmt.Println("You may be offline or the GitHub API is unavailable.")
		return nil
	}

	fmt.Printf("Latest version:  v%s\n", latestVersion)

	if currentVersion.GTE(latestVersion) && currentVersion.String() != "0.0.0" {
		fmt.Println("\nYou are on the latest version!")
		return nil
	}

	if latestVersion.LT(currentVersion) {
		fmt.Println("\nYou are on a newer version than the latest release!")
		return nil
	}

	fmt.Printf("\nNew version available: v%s\n", latestVersion)

	if checkOnly {
		return nil
	}

	if !autoUpdate {
		fmt.Print("\nWould you like to update? (y/N): ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		if response != "y" && response != "Y" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	fmt.Println("\nDownloading update...")

	installPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find executable path: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "lissh-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	filename, err := downloadVersion(latestVersion, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	newBinaryPath := filepath.Join(tmpDir, filename)
	if err := os.Chmod(newBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	newPath := installPath + ".new"
	if err := copyFile(newBinaryPath, newPath); err != nil {
		return fmt.Errorf("failed to prepare update: %w", err)
	}

	binDir := filepath.Dir(installPath)
	helperPath := filepath.Join(binDir, ".lissh_update_helper")

	helperScript := fmt.Sprintf(`#!/bin/bash
sleep 0.5
rm -f '%s'
mv '%s' '%s'
chmod 755 '%s'
rm -f '%s'
exec '%s'
`, installPath, newPath, installPath, installPath, helperPath, installPath)

	if err := os.WriteFile(helperPath, []byte(helperScript), 0755); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("failed to create helper: %w", err)
	}

	procAttr := &syscall.ProcAttr{
		Dir: binDir,
		Env: os.Environ(),
		Sys: &syscall.SysProcAttr{Setsid: true},
	}

	_, err = syscall.ForkExec(helperPath, []string{"lissh-update-helper"}, procAttr)
	if err != nil {
		os.Remove(newPath)
		os.Remove(helperPath)
		return fmt.Errorf("failed to start update helper: %w", err)
	}

	fmt.Printf("\nSuccessfully updated to v%s!\n", latestVersion)
	fmt.Println("Run 'lissh --version' to verify.")

	return nil
}

func downloadVersion(v semver.Version, tmpDir string) (string, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	osTitle := strings.ToUpper(goos[:1]) + goos[1:]
	filename := fmt.Sprintf("lissh_%s_%s.%s", osTitle, arch, ext)
	url := fmt.Sprintf("https://github.com/tesh254/lissh/releases/download/v%s/%s", v, filename)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filepath.Join(tmpDir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filename, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return dstFile.Sync()
}
