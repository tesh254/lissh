package history

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/pkg/style"
)

var historyDB *storage.DB

func NewHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "View SSH connection history",
		Long:  `Track and view your SSH connection sessions.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent connections",
		RunE:  listHistory,
	}
	listCmd.Flags().Int("limit", 50, "Number of records to show")
	listCmd.Flags().Int("offset", 0, "Offset for pagination")

	hostCmd := &cobra.Command{
		Use:   "host [id]",
		Short: "Show connection history for a specific host",
		Args:  cobra.ExactArgs(1),
		RunE:  hostHistory,
	}
	hostCmd.Flags().Int("limit", 10, "Number of records to show")

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear connection history",
		RunE:  clearHistory,
	}
	clearCmd.Flags().Int64("host", 0, "Clear history for specific host only")
	clearCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	cmd.AddCommand(listCmd, hostCmd, clearCmd)

	return cmd
}

func SetDB(db *storage.DB) {
	historyDB = db
}

func listHistory(cmd *cobra.Command, _ []string) error {
	if historyDB == nil {
		return fmt.Errorf("database not initialized")
	}
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	histories, err := historyDB.ListHistory(limit, offset)
	if err != nil {
		return fmt.Errorf("failed to list history: %w", err)
	}

	if len(histories) == 0 {
		fmt.Println(style.Dim.Render("No connection history yet"))
		return nil
	}

	fmt.Println(style.Header.Render(" ID   Host                  Started               Duration      "))
	fmt.Println(style.Subtle.Render(" ──────────────────────────────────────────────────────────────────"))
	for _, h := range histories {
		host := h.Hostname
		if h.Alias != nil && *h.Alias != "" {
			host = *h.Alias
		}
		duration := style.Warning.Render("ongoing")
		if h.DurationSeconds != nil {
			duration = formatDuration(*h.DurationSeconds)
		}
		fmt.Printf(" %s %s %s %s\n",
			style.IDStyle.Render(fmt.Sprintf("%d", h.ID)),
			style.Bold.Render(truncate(host, 20)),
			style.Subtle.Render(h.StartedAt.Format("2006-01-02 15:04")),
			style.Info.Render(duration),
		)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-2] + ".."
	}
	return s
}

func hostHistory(cmd *cobra.Command, args []string) error {
	if historyDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	host, err := historyDB.GetHostByID(id)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}
	if host == nil {
		return fmt.Errorf("host not found")
	}

	limit, _ := cmd.Flags().GetInt("limit")

	histories, err := historyDB.ListHistoryByHost(id, limit)
	if err != nil {
		return fmt.Errorf("failed to list history: %w", err)
	}

	alias := host.Hostname
	if host.Alias != nil && *host.Alias != "" {
		alias = *host.Alias
	}

	fmt.Printf("Connection history for: %s (%s)\n\n", alias, host.Hostname)

	if len(histories) == 0 {
		fmt.Println("No connection history for this host")
		return nil
	}

	fmt.Printf("%-5s %-20s %-15s\n", "ID", "Started", "Duration")
	fmt.Println("-----------------------------------------")
	for _, h := range histories {
		duration := "ongoing"
		if h.DurationSeconds != nil {
			duration = formatDuration(*h.DurationSeconds)
		}
		fmt.Printf("%-5d %-20s %-15s\n", h.ID, h.StartedAt.Format("2006-01-02 15:04"), duration)
	}

	return nil
}

func clearHistory(cmd *cobra.Command, _ []string) error {
	if historyDB == nil {
		return fmt.Errorf("database not initialized")
	}
	hostID, _ := cmd.Flags().GetInt64("host")
	confirm, _ := cmd.Flags().GetBool("confirm")

	if !confirm {
		if hostID > 0 {
			fmt.Print("Clear all history for this host? (y/N): ")
		} else {
			fmt.Print("Clear ALL connection history? This cannot be undone. (y/N): ")
		}
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	if hostID > 0 {
		if err := historyDB.DeleteHistoryByHost(hostID); err != nil {
			return fmt.Errorf("failed to clear history: %w", err)
		}
		fmt.Println("History cleared for host")
	} else {
		histories, _ := historyDB.ListHistory(10000, 0)
		for _, h := range histories {
			if err := historyDB.DeleteHistory(h.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete history entry %d: %v\n", h.ID, err)
			}
		}
		fmt.Println("All history cleared")
	}

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
	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}
