package conn

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/pkg/style"
)

var connDB *storage.DB

func NewConnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conn [alias]",
		Short: "Connect to an SSH host",
		Long:  `Connect to an SSH host by alias or ID. Will prompt for user and port if needed.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runConn,
	}

	cmd.Flags().Int64P("id", "i", 0, "Connect to host by ID")

	return cmd
}

func SetDB(db *storage.DB) {
	connDB = db
}

func runConn(cmd *cobra.Command, args []string) error {
	if connDB == nil {
		return fmt.Errorf("database not initialized")
	}

	var host *storage.Host
	var err error

	switch {
	case cmd.Flags().Changed("id"):
		idVal, _ := cmd.Flags().GetInt64("id")
		host, err = connDB.GetHostByID(idVal)
		if err != nil {
			return fmt.Errorf("failed to get host: %w", err)
		}
		if host == nil {
			return fmt.Errorf("host not found with ID %d", idVal)
		}
	case len(args) > 0:
		target := args[0]
		host, err = connDB.GetHostByHostname(target)
		if err != nil {
			return fmt.Errorf("failed to find host: %w", err)
		}
		if host == nil {
			hosts, err := connDB.SearchHosts(target)
			if err == nil && len(hosts) == 1 {
				host = hosts[0]
			}
		}
		if host == nil {
			return fmt.Errorf("host not found: %s", target)
		}
	default:
		return cmd.Help()
	}

	if host.IsInactive {
		return fmt.Errorf("host %s is marked inactive", host.Hostname)
	}

	user := getUser(host)
	port := getPort(host)

	target := host.Hostname
	if host.IPAddress != nil && *host.IPAddress != "" {
		target = *host.IPAddress
	}

	fullTarget := fmt.Sprintf("%s@%s:%d", user, target, port)

	session, err := connDB.StartSession(host.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start session tracking: %v\n", err)
	}

	sshCmd := exec.Command("ssh", fullTarget)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	err = sshCmd.Run()

	if session != nil {
		if endErr := connDB.EndSession(session.ID); endErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to end session tracking: %v\n", endErr)
		}
	}

	return err
}

func getUser(host *storage.Host) string {
	currentUser := ""
	if host.User != nil && *host.User != "" {
		currentUser = *host.User
	}

	fmt.Printf("  %s %s\n", style.Info.Render("Host:"), style.Bold.Render(host.Hostname))
	if host.Alias != nil && *host.Alias != "" {
		fmt.Printf("  %s %s\n", style.Info.Render("Alias:"), style.OK.Render(*host.Alias))
	}

	if currentUser == "" {
		fmt.Printf("  %s ", style.Warning.Render("User:"))
		var user string
		if _, err := fmt.Scanln(&user); err != nil {
			user = "root"
		}
		if user == "" {
			user = "root"
		}
		return user
	}

	fmt.Printf("  %s [%s]: ", style.Info.Render("User"), style.Bold.Render(currentUser))
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return currentUser
	}
	if response == "" {
		return currentUser
	}
	return response
}

func getPort(host *storage.Host) int {
	currentPort := host.Port
	if currentPort == 0 {
		currentPort = 22
	}

	fmt.Printf("  %s [%d]: ", style.Info.Render("Port"), currentPort)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return currentPort
	}
	if response == "" {
		return currentPort
	}

	port, err := strconv.Atoi(response)
	if err != nil || port < 1 || port > 65535 {
		fmt.Printf("  %s Using default port %d\n", style.Warning.Render("Invalid port:"), currentPort)
		return currentPort
	}

	return port
}
