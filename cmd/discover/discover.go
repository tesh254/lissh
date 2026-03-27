package discover

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/discovery"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/pkg/style"
)

var discoverDB *storage.DB

func NewDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover SSH hosts from your system",
		Long: `Scans known_hosts, SSH config, and other locations to find
SSH-accessible hosts on your system.`,
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Discover and list new hosts",
		RunE:  runDiscovery,
	}
	runCmd.Flags().Bool("all", false, "Show all discovered hosts including already known ones")
	runCmd.Flags().Bool("dry-run", false, "Show what would be added without adding")

	reviewCmd := &cobra.Command{
		Use:   "review",
		Short: "Step through discovered hosts one by one",
		RunE:  reviewHosts,
	}

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Infer usernames from shell history",
		Long:  `Parses shell history to find SSH connections and updates host records with usernames.`,
		RunE:  discoverUsers,
	}
	usersCmd.Flags().Bool("dry-run", false, "Show what would be updated without updating")

	cmd.AddCommand(runCmd, reviewCmd, usersCmd)

	return cmd
}

func SetDB(db *storage.DB) {
	discoverDB = db
}

func discoverUsers(cmd *cobra.Command, _ []string) error {
	if discoverDB == nil {
		return fmt.Errorf("database not initialized")
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	userMap := parseShellHistory()

	if len(userMap) == 0 {
		fmt.Println(style.Subtle.Render("No SSH connections found in shell history"))
		return nil
	}

	fmt.Printf("  %s Found %d user@host patterns in history\n\n", style.Header.Render("Scanning..."), len(userMap))

	hosts, err := discoverDB.ListHosts(false)
	if err != nil {
		return fmt.Errorf("failed to list hosts: %w", err)
	}

	updated := 0
	for _, h := range hosts {
		if h.User != nil && *h.User != "" {
			continue
		}
		if user, ok := userMap[h.Hostname]; ok {
			if dryRun {
				fmt.Printf("  %s %s -> user would be set to %s\n", style.Warning.Render("[~]"), h.Hostname, style.Bold.Render(user))
			} else {
				if err := discoverDB.UpdateHost(h.ID, nil, nil, nil, &user); err != nil {
					fmt.Printf("  %s %s (failed: %v)\n", style.Error.Render("[X]"), h.Hostname, err)
				} else {
					fmt.Printf("  %s %s -> user set to %s\n", style.OK.Render("[+]"), h.Hostname, style.Bold.Render(user))
				}
			}
			updated++
		}
	}

	if dryRun {
		fmt.Printf("\n  %s %d hosts would be updated\n", style.Warning.Render("Dry run:"), updated)
	} else {
		fmt.Printf("\n  %s %d hosts updated with usernames\n", style.Success.Render("Done!"), updated)
	}
	return nil
}

func parseShellHistory() map[string]string {
	results := make(map[string]string)

	historyFiles := []string{
		expandPath("~/.bash_history"),
		expandPath("~/.zsh_history"),
		expandPath("~/.history"),
	}

	sshPattern := []string{
		"ssh ",
		"ssh -l ",
		"ssh -p ",
		"scp ",
		"sftp ",
	}

	for _, histFile := range historyFiles {
		data, err := os.ReadFile(histFile)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			for _, pattern := range sshPattern {
				if idx := strings.Index(line, pattern); idx >= 0 {
					rest := line[idx+len(pattern):]
					user, host := parseSSHCommand(rest)
					if host != "" && user != "" {
						results[host] = user
					}
					break
				}
			}
		}
	}

	return results
}

func parseSSHCommand(cmd string) (user, host string) {
	parts := strings.Fields(cmd)
	for i, p := range parts {
		if p == "-p" && i+1 < len(parts) {
			port := parts[i+1]
			if _, err := fmt.Sscanf(port, "%d", &port); err == nil {
				cmd = strings.Replace(cmd, "-p "+port, "", 1)
			}
		}
	}

	cmd = strings.TrimSpace(cmd)
	cmd = strings.Split(cmd, " -- ")[0]

	if idx := strings.Index(cmd, "@"); idx > 0 {
		user = cmd[:idx]
		host = cmd[idx+1:]
		host = strings.Split(host, " ")[0]
		host = strings.Split(host, ":")[0]
		return
	}

	host = strings.Fields(cmd)[0]
	host = strings.Split(host, ":")[0]
	return
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

func runDiscovery(cmd *cobra.Command, _ []string) error {
	if discoverDB == nil {
		return fmt.Errorf("database not initialized")
	}
	showAll, _ := cmd.Flags().GetBool("all")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	disc := discovery.New(discovery.DefaultConfig())

	hosts, err := disc.DiscoverAll()
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if len(hosts) == 0 {
		fmt.Println(style.Subtle.Render("No new hosts discovered"))
		return nil
	}

	fmt.Printf("  %s %d\n", style.Header.Render("Discovered"), len(hosts))
	fmt.Println()

	newHosts := []*discovery.DiscoveredHost{}
	for _, h := range hosts {
		exists, _ := discoverDB.HostExists(h.Hostname)
		if exists {
			if showAll {
				fmt.Printf("  %s %s\n", style.Inactive.Render("[skip]"), style.Subtle.Render(h.Hostname+" (already known)"))
			}
			continue
		}
		newHosts = append(newHosts, h)
		userInfo := ""
		if h.User != "" {
			userInfo = fmt.Sprintf(" %s@", h.User)
		}
		fmt.Printf("  %s %s%s %s %s\n", style.OK.Render("[+]"), style.Bold.Render(h.Hostname), style.Info.Render(fmt.Sprintf("%s%s", userInfo, h.IPAddress)), style.Subtle.Render("from"), style.Dim.Render(h.Source))
	}

	if dryRun {
		fmt.Printf("\n  %s %d hosts would be added\n", style.Warning.Render("Dry run:"), len(newHosts))
		return nil
	}

	if len(newHosts) == 0 {
		fmt.Println(style.Subtle.Render("\nNo new hosts to add"))
		return nil
	}

	added := 0
	for _, h := range newHosts {
		_, err := discoverDB.CreateHost(storage.CreateHostInput{
			Hostname:  h.Hostname,
			IPAddress: &h.IPAddress,
			User:      &h.User,
			Port:      h.Port,
			Source:    h.Source,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s Failed to add %s: %v\n", style.Error.Render("!"), h.Hostname, err)
			continue
		}
		added++
	}

	fmt.Printf("\n  %s %d new hosts added\n", style.Success.Render("Done!"), added)
	fmt.Println("Run 'lissh hosts list' to see all hosts")
	fmt.Println("Run 'lissh discover review' to label and organize hosts")

	return nil
}

func reviewHosts(cmd *cobra.Command, _ []string) error {
	if discoverDB == nil {
		return fmt.Errorf("database not initialized")
	}
	hosts, err := discoverDB.ListHosts(true)
	if err != nil {
		return fmt.Errorf("failed to list hosts: %w", err)
	}

	if len(hosts) == 0 {
		fmt.Println("No hosts to review. Run 'lissh discover' first.")
		return nil
	}

	fmt.Println("SSH on a leash - Host Review")
	fmt.Println("============================")
	fmt.Println("For each host, you can:")
	fmt.Println("  [a] Alias - Set a friendly name")
	fmt.Println("  [n] Notes - Add notes/labels")
	fmt.Println("  [k] Key   - Associate an SSH key")
	fmt.Println("  [i] Inactivate - Mark as inactive")
	fmt.Println("  [s] Skip  - Move to next host")
	fmt.Println("  [q] Quit  - Exit review mode")
	fmt.Println()

	for i, h := range hosts {
		fmt.Printf("[%d/%d] Host: %s\n", i+1, len(hosts), h.Hostname)
		if h.Alias != nil {
			fmt.Printf("  Alias: %s\n", *h.Alias)
		}
		if h.Notes != nil {
			fmt.Printf("  Notes: %s\n", *h.Notes)
		}
		fmt.Printf("  Status: %s\n", map[bool]string{true: "inactive", false: "active"}[h.IsInactive])

		var response string
		fmt.Print("Action ([a]/[n]/[k]/[i]/[s]/[q]): ")
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}

		switch response {
		case "q":
			fmt.Println("Exiting review mode")
			return nil
		case "i":
			if err := discoverDB.MarkHostInactive(h.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				fmt.Println("  Marked as inactive")
			}
		case "s":
			continue
		case "a":
			fmt.Print("  Enter alias: ")
			var alias string
			fmt.Scanln(&alias)
			if alias != "" {
				if err := discoverDB.UpdateHost(h.ID, &alias, nil, nil, nil); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				} else {
					fmt.Printf("  Alias set to: %s\n", alias)
				}
			}
		case "n":
			fmt.Print("  Enter notes: ")
			var notes string
			fmt.Scanln(&notes)
			if notes != "" {
				if err := discoverDB.UpdateHost(h.ID, nil, &notes, nil, nil); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				} else {
					fmt.Println("  Notes updated")
				}
			}
		case "k":
			keys, _ := discoverDB.ListSSHKeys()
			if len(keys) == 0 {
				fmt.Println("  No SSH keys found. Use 'lissh keys create' to add one.")
			} else {
				fmt.Println("  Available keys:")
				for _, k := range keys {
					fmt.Printf("    [%d] %s (%s)\n", k.ID, k.Name, k.KeyType)
				}
				fmt.Print("  Enter key ID (or 0 to skip): ")
				var keyID int64
				fmt.Scanln(&keyID)
				if keyID > 0 {
					if err := discoverDB.UpdateHost(h.ID, nil, nil, &keyID, nil); err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					} else {
						fmt.Println("  SSH key associated")
					}
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("Review complete!")
	return nil
}
