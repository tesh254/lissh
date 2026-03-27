package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/cmd/config"
	"github.com/wcrg/lissh/cmd/discover"
	"github.com/wcrg/lissh/cmd/history"
	"github.com/wcrg/lissh/cmd/hosts"
	"github.com/wcrg/lissh/cmd/keys"
	"github.com/wcrg/lissh/internal/storage"
)

var (
	db     *storage.DB
	dbPath string
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "lissh",
		Short: "lissh - SSH on a leash",
		Long: `lissh keeps your SSH hosts organized and on a leash.

Quickly list and search hosts, assign friendly aliases, track your
connection history, manage SSH keys, and tweak SSH settings - all
without manually editing config files.`,
		RunE:                  runRoot,
		DisableAutoGenTag:     true,
		DisableFlagsInUseLine: true,
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "Path to lissh database (default: ~/.lissh/lissh.db)")
	rootCmd.Flags().Int64VarP(new(int64), "id", "i", 0, "Connect to host by ID")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "lissh" && (len(args) > 0 || cmd.Flags().Changed("id")) {
			return nil
		}
		var err error
		db, err = storage.New(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		hosts.SetDB(db)
		discover.SetDB(db)
		keys.SetDB(db)
		history.SetDB(db)
		return nil
	}
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if db != nil {
			return db.Close()
		}
		return nil
	}

	rootCmd.AddCommand(hosts.NewHostsCmd())
	rootCmd.AddCommand(discover.NewDiscoverCmd())
	rootCmd.AddCommand(keys.NewKeysCmd())
	rootCmd.AddCommand(history.NewHistoryCmd())
	rootCmd.AddCommand(config.NewConfigCmd())

	return rootCmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	dbInit, err := storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer dbInit.Close()

	var host *storage.Host

	switch {
	case cmd.Flags().Changed("id"):
		idVal, _ := cmd.Flags().GetInt64("id")
		host, err = dbInit.GetHostByID(idVal)
		if err != nil {
			return fmt.Errorf("failed to get host: %w", err)
		}
		if host == nil {
			return fmt.Errorf("host not found")
		}
	case len(args) > 0:
		target := args[0]
		host, err = dbInit.GetHostByHostname(target)
		if err != nil {
			return fmt.Errorf("failed to find host: %w", err)
		}
		if host == nil {
			hosts, err := dbInit.SearchHosts(target)
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

	session, err := dbInit.StartSession(host.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start session tracking: %v\n", err)
	}

	target := host.Hostname
	if host.IPAddress != nil && *host.IPAddress != "" {
		target = *host.IPAddress
	}
	user := ""
	if host.User != nil && *host.User != "" {
		user = *host.User + "@"
	}
	if host.Port != 22 {
		target = fmt.Sprintf("%s:%d", target, host.Port)
	}

	fullTarget := user + target

	sshCmd := exec.Command("ssh", fullTarget)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	err = sshCmd.Run()

	if session != nil {
		if endErr := dbInit.EndSession(session.ID); endErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to end session tracking: %v\n", endErr)
		}
	}

	return err
}

func Execute() error {
	c := NewRootCmd()
	if err := c.Execute(); err != nil {
		return err
	}
	return nil
}

func NewDefaultDB() (*storage.DB, error) {
	return storage.New("")
}
