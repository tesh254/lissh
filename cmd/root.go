package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/cmd/actions"
	"github.com/wcrg/lissh/cmd/config"
	"github.com/wcrg/lissh/cmd/conn"
	"github.com/wcrg/lissh/cmd/discover"
	"github.com/wcrg/lissh/cmd/history"
	"github.com/wcrg/lissh/cmd/hosts"
	"github.com/wcrg/lissh/cmd/keys"
	"github.com/wcrg/lissh/cmd/update"
	"github.com/wcrg/lissh/internal/assets"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/internal/version"
)

var (
	db     *storage.DB
	dbPath string
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                   "lissh",
		Short:                 "SSH on a leash",
		Long:                  assets.LOGO + "\nKeeps your SSH hosts organized and on a leash.\n\nQuickly list and search hosts, assign friendly aliases, track your\nconnection history, manage SSH keys, and tweak SSH settings - all\nwithout manually editing config files.\n\nRun 'lissh update --check' to check for updates or 'lissh update --install' to update.",
		DisableAutoGenTag:     true,
		DisableFlagsInUseLine: true,
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "Path to lissh database (default: ~/.lissh/lissh.db)")
	rootCmd.Flags().Bool("check-update", false, "Check for updates")
	rootCmd.Flags().Bool("version", false, "Show version information")
	rootCmd.Flags().Bool("logo", false, "Show logo")
	rootCmd.SetVersionTemplate("lissh {{.Version}}\n")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("logo") {
			fmt.Print(assets.LOGO)
			os.Exit(0)
			return nil
		}
		if cmd.Flags().Changed("version") {
			fmt.Printf("%s lissh %s\n", assets.LOGO, version.String())
			os.Exit(0)
			return nil
		}
		if cmd.Flags().Changed("check-update") {
			checkForUpdate()
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
		conn.SetDB(db)
		actions.SetDB(db)
		return nil
	}
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if db != nil {
			return db.Close()
		}
		return nil
	}

	rootCmd.AddCommand(hosts.NewHostsCmd())
	rootCmd.AddCommand(conn.NewConnCmd())
	rootCmd.AddCommand(discover.NewDiscoverCmd())
	rootCmd.AddCommand(keys.NewKeysCmd())
	rootCmd.AddCommand(history.NewHistoryCmd())
	rootCmd.AddCommand(config.NewConfigCmd())
	rootCmd.AddCommand(update.NewUpdateCmd())
	rootCmd.AddCommand(actions.NewActionsCmd())

	return rootCmd
}

func checkForUpdate() {
	currentVersion := version.Get()
	latestVersion, err := version.CheckForUpdate()
	if err != nil {
		return
	}

	if latestVersion.GT(currentVersion) {
		fmt.Printf("\n  New version available: v%s\n", latestVersion)
		fmt.Println("  Run 'lissh update --install' to update.")
	}
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
