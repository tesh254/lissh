package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

	if latestVersion.GTE(currentVersion) && currentVersion.String() != "0.0.0" {
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
	defer os.RemoveAll(tmpDir)

	filename, err := downloadVersion(latestVersion, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, filename)
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	fmt.Printf("Installing to: %s\n", installPath)

	if err := os.Rename(binaryPath, installPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w (try running with sudo)", err)
	}

	fmt.Printf("\nSuccessfully updated to v%s!\n", latestVersion)
	fmt.Printf("Run 'lissh --version' to verify.\n")

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
