package hosts

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/pkg/style"
)

var hostsDB *storage.DB

func NewHostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "Manage SSH hosts",
		Long:  `List, search, edit, and connect to your SSH hosts.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all hosts",
		RunE:  listHosts,
	}
	listCmd.Flags().Bool("all", false, "Show inactive hosts too")

	searchCmd := &cobra.Command{
		Use:   "search [term]",
		Short: "Search hosts by name, alias, or notes",
		Args:  cobra.ExactArgs(1),
		RunE:  searchHosts,
	}

	infoCmd := &cobra.Command{
		Use:   "info [id]",
		Short: "Show detailed info about a host",
		Args:  cobra.ExactArgs(1),
		RunE:  showHostInfo,
	}

	editCmd := &cobra.Command{
		Use:   "edit [id]",
		Short: "Edit a host's alias, notes, or SSH key",
		Args:  cobra.ExactArgs(1),
		RunE:  editHost,
	}
	editCmd.Flags().String("alias", "", "Set host alias")
	editCmd.Flags().String("notes", "", "Set host notes")
	editCmd.Flags().Int64("key-id", 0, "Associate SSH key ID")
	editCmd.Flags().Int("port", 0, "Set port")

	connectCmd := &cobra.Command{
		Use:   "connect [id]",
		Short: "Connect to a host via SSH",
		Args:  cobra.ExactArgs(1),
		RunE:  connectToHost,
	}

	inactivateCmd := &cobra.Command{
		Use:   "inactivate [id]",
		Short: "Mark a host as inactive (keeps history)",
		Args:  cobra.ExactArgs(1),
		RunE:  inactivateHost,
	}

	removeCmd := &cobra.Command{
		Use:   "remove [id]",
		Short: "Permanently remove a host and its history",
		Args:  cobra.ExactArgs(1),
		RunE:  removeHost,
	}
	removeCmd.Flags().Bool("dry-run", false, "Show what would be removed")
	removeCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	pruneCmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove all inactive hosts",
		RunE:  pruneHosts,
	}
	pruneCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	cmd.AddCommand(listCmd, searchCmd, infoCmd, editCmd, connectCmd, inactivateCmd, removeCmd, pruneCmd)

	return cmd
}

func SetDB(db *storage.DB) {
	hostsDB = db
}

func listHosts(cmd *cobra.Command, _ []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	showInactive, _ := cmd.Flags().GetBool("all")
	hosts, err := hostsDB.ListHosts(showInactive)
	if err != nil {
		return fmt.Errorf("failed to list hosts: %w", err)
	}

	if len(hosts) == 0 {
		fmt.Println(style.Dim.Render("No hosts found. Run "), style.Info.Render("lissh discover"), style.Dim.Render(" to find hosts."))
		return nil
	}

	sortHostsByIP(hosts)

	fmt.Printf("  %-4s  %-22s  %-28s  %-18s  %s\n",
		style.Header.Render("ID"),
		style.Header.Render("Alias"),
		style.Header.Render("Hostname"),
		style.Header.Render("User@Port"),
		style.Header.Render("Status"))
	fmt.Println(style.Subtle.Render("  ───────────────────────────────────────────────────────────────────────────────────────────────────"))

	for _, h := range hosts {
		alias := h.Hostname
		if h.Alias != nil && *h.Alias != "" {
			alias = *h.Alias
		}
		userPort := fmt.Sprintf("%d", h.Port)
		userStr := ""
		if h.User != nil && *h.User != "" {
			userStr = *h.User
			userPort = fmt.Sprintf("%s@%d", userStr, h.Port)
		}
		statusText := "active"
		statusStyle := style.OK
		if h.IsInactive {
			statusText = "inactive"
			statusStyle = style.Inactive
		}

		fmt.Printf("  %-4d  %-22s  %-28s  %-18s  %s\n",
			h.ID,
			truncate(alias, 22),
			truncate(h.Hostname, 28),
			truncate(userPort, 18),
			statusStyle.Render(statusText),
		)
	}

	return nil
}

func sortHostsByIP(hosts []*storage.Host) {
	sort.Slice(hosts, func(i, j int) bool {
		hi, hj := hosts[i], hosts[j]

		pi := parseIP(hi.Hostname)
		pj := parseIP(hj.Hostname)

		if pi == nil {
			pi = parseIP(getStringPtr(hi.IPAddress))
		}
		if pj == nil {
			pj = parseIP(getStringPtr(hj.IPAddress))
		}

		if pi == nil && pj == nil {
			return hi.Hostname < hj.Hostname
		}
		if pi == nil {
			return false
		}
		if pj == nil {
			return true
		}

		for k := 0; k < len(pi); k++ {
			if pi[k] != pj[k] {
				return pi[k] < pj[k]
			}
		}
		return false
	})
}

func parseIP(s string) []int {
	if s == "" {
		return nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		parts := strings.Split(s, ".")
		if len(parts) != 4 {
			return nil
		}
		result := make([]int, 4)
		for i, p := range parts {
			fmt.Sscanf(p, "%d", &result[i])
		}
		return result
	}
	ip4 := ip.To4()
	if ip4 != nil {
		return []int{int(ip4[0]), int(ip4[1]), int(ip4[2]), int(ip4[3])}
	}
	return nil
}

func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-2] + ".."
	}
	return s
}

func searchHosts(cmd *cobra.Command, args []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	term := args[0]
	hosts, err := hostsDB.SearchHosts(term)
	if err != nil {
		return fmt.Errorf("failed to search hosts: %w", err)
	}

	if len(hosts) == 0 {
		fmt.Printf("%s No hosts found matching %s\n", style.Warning.Render("?"), style.Bold.Render(term))
		return nil
	}

	fmt.Println(style.Header.Render(" ID   Alias                 Hostname                  User@Port        "))
	fmt.Println(style.Subtle.Render(" ───────────────────────────────────────────────────────────────────────"))
	for _, h := range hosts {
		alias := h.Hostname
		if h.Alias != nil && *h.Alias != "" {
			alias = *h.Alias
		}
		userPort := fmt.Sprintf("%d", h.Port)
		userStr := ""
		if h.User != nil && *h.User != "" {
			userStr = *h.User
			userPort = fmt.Sprintf("%s@%d", userStr, h.Port)
		}
		fmt.Printf(" %s %s %s %s\n",
			style.IDStyle.Render(fmt.Sprintf("%d", h.ID)),
			style.Bold.Render(truncate(alias, 20)),
			style.Info.Render(truncate(h.Hostname, 25)),
			style.UserPortColor(userStr).Render(truncate(userPort, 15)),
		)
	}

	return nil
}

func showHostInfo(cmd *cobra.Command, args []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	host, err := hostsDB.GetHostByID(id)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}
	if host == nil {
		return fmt.Errorf("host not found")
	}

	fmt.Println()
	fmt.Printf("  %s %d\n", style.Info.Render("ID:"), host.ID)
	fmt.Printf("  %s %s\n", style.Info.Render("Hostname:"), style.Bold.Render(host.Hostname))
	if host.Alias != nil {
		fmt.Printf("  %s %s\n", style.Info.Render("Alias:"), style.OK.Render(*host.Alias))
	}
	if host.IPAddress != nil {
		fmt.Printf("  %s %s\n", style.Info.Render("IP Address:"), *host.IPAddress)
	}
	if host.User != nil && *host.User != "" {
		fmt.Printf("  %s %s\n", style.Info.Render("User:"), style.Bold.Render(*host.User))
	}
	fmt.Printf("  %s %d\n", style.Info.Render("Port:"), host.Port)
	fmt.Printf("  %s %s\n", style.Info.Render("Source:"), style.Subtle.Render(host.Source))
	if host.Notes != nil && *host.Notes != "" {
		fmt.Printf("  %s %s\n", style.Info.Render("Notes:"), *host.Notes)
	}
	fmt.Printf("  %s %s\n", style.Info.Render("Discovered:"), style.Subtle.Render(host.DiscoveredAt.Format("2006-01-02 15:04:05")))

	statusText := "active"
	statusStyle := style.OK
	if host.IsInactive {
		statusText = "inactive"
		statusStyle = style.Inactive
	}
	fmt.Printf("  %s %s\n", style.Info.Render("Status:"), statusStyle.Render(statusText))

	fmt.Println()
	totalSessions, totalDuration, lastAccessed, _ := hostsDB.GetHostAccessStats(id)
	fmt.Printf("  %s %d\n", style.Info.Render("Total Sessions:"), totalSessions)
	fmt.Printf("  %s %s\n", style.Info.Render("Total Time:"), formatDuration(totalDuration))
	if lastAccessed != nil {
		fmt.Printf("Last Accessed:  %s\n", lastAccessed.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func editHost(cmd *cobra.Command, args []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	alias, _ := cmd.Flags().GetString("alias")
	notes, _ := cmd.Flags().GetString("notes")
	sshKeyID, _ := cmd.Flags().GetInt64("key-id")
	port, _ := cmd.Flags().GetInt("port")

	var aliasPtr, notesPtr *string
	if cmd.Flags().Changed("alias") {
		aliasPtr = &alias
	}
	if cmd.Flags().Changed("notes") {
		notesPtr = &notes
	}
	var sshKeyPtr *int64
	if cmd.Flags().Changed("key-id") {
		sshKeyPtr = &sshKeyID
	}

	if aliasPtr == nil && notesPtr == nil && sshKeyPtr == nil && !cmd.Flags().Changed("port") {
		return fmt.Errorf("no changes specified (use --alias, --notes, --key-id, or --port)")
	}

	if cmd.Flags().Changed("port") {
		if err := hostsDB.UpdateHostPort(id, port); err != nil {
			return fmt.Errorf("failed to update host port: %w", err)
		}
	}

	if err := hostsDB.UpdateHost(id, aliasPtr, notesPtr, sshKeyPtr, nil); err != nil {
		return fmt.Errorf("failed to update host: %w", err)
	}

	fmt.Println("Host updated successfully")
	return nil
}

func connectToHost(cmd *cobra.Command, args []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	host, err := hostsDB.GetHostByID(id)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}
	if host == nil {
		return fmt.Errorf("host not found")
	}
	if host.IsInactive {
		return fmt.Errorf("host is marked inactive")
	}

	session, err := hostsDB.StartSession(host.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start session tracking: %v\n", err)
	}

	target := host.Hostname
	if host.IPAddress != nil && *host.IPAddress != "" {
		target = *host.IPAddress
	}
	if host.Port != 22 {
		target = fmt.Sprintf("%s:%d", target, host.Port)
	}

	sshCmd := exec.Command("ssh", target)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	err = sshCmd.Run()

	if session != nil {
		if endErr := hostsDB.EndSession(session.ID); endErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to end session tracking: %v\n", endErr)
		}
	}

	return err
}

func inactivateHost(cmd *cobra.Command, args []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	if err := hostsDB.MarkHostInactive(id); err != nil {
		return fmt.Errorf("failed to inactivate host: %w", err)
	}

	fmt.Println("Host marked as inactive")
	return nil
}

func removeHost(cmd *cobra.Command, args []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		host, _ := hostsDB.GetHostByID(id)
		if host == nil {
			return fmt.Errorf("host not found")
		}
		fmt.Printf("Would remove host: %s (ID: %d)\n", host.Hostname, id)
		fmt.Println("This would also remove all history for this host.")
		return nil
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		fmt.Print("This will permanently remove the host and all its history. Continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	if err := hostsDB.DeleteHistoryByHost(id); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete history: %v\n", err)
	}
	if err := hostsDB.DeleteHost(id); err != nil {
		return fmt.Errorf("failed to delete host: %w", err)
	}

	fmt.Println("Host removed successfully")
	return nil
}

func pruneHosts(cmd *cobra.Command, _ []string) error {
	if hostsDB == nil {
		return fmt.Errorf("database not initialized")
	}
	hosts, err := hostsDB.ListHosts(false)
	if err != nil {
		return fmt.Errorf("failed to list hosts: %w", err)
	}

	var inactive []*storage.Host
	for _, h := range hosts {
		if h.IsInactive {
			inactive = append(inactive, h)
		}
	}

	if len(inactive) == 0 {
		fmt.Println("No inactive hosts to prune")
		return nil
	}

	fmt.Printf("Found %d inactive hosts:\n", len(inactive))
	for _, h := range inactive {
		fmt.Printf("  - %s (ID: %d)\n", h.Hostname, h.ID)
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		fmt.Print("\nRemove all inactive hosts and their history? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	ids := make([]int64, len(inactive))
	for i, h := range inactive {
		ids[i] = h.ID
	}

	if err := hostsDB.BulkMarkInactive(ids); err != nil {
		return fmt.Errorf("failed to prune hosts: %w", err)
	}

	fmt.Println("Inactive hosts pruned successfully")
	return nil
}

func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
