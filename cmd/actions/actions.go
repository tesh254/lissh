package actions

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/pkg/style"
)

var actionsDB *storage.DB

var variableRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

func NewActionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "Manage and run remote actions",
		Long:  `Manage actions that can be executed on remote hosts via SSH.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all actions",
		RunE:  listActions,
	}

	infoCmd := &cobra.Command{
		Use:   "info [name]",
		Short: "Show detailed info about an action",
		Args:  cobra.ExactArgs(1),
		RunE:  showActionInfo,
	}

	addCmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new action",
		Args:  cobra.ExactArgs(1),
		RunE:  addAction,
	}
	addCmd.Flags().String("description", "", "Action description")
	addCmd.Flags().String("command", "", "Command template to execute")
	addCmd.Flags().String("host-alias", "", "Comma-separated host aliases to bind (e.g., sigma-dev,sigma-beta)")
	_ = addCmd.MarkFlagRequired("command")

	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit an existing action",
		Args:  cobra.ExactArgs(1),
		RunE:  editAction,
	}
	editCmd.Flags().String("description", "", "Action description")
	editCmd.Flags().String("command", "", "Command template to execute")
	editCmd.Flags().String("host-alias", "", "Comma-separated host aliases to bind (e.g., sigma-dev,sigma-beta)")

	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete an action",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteAction,
	}
	deleteCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	runCmd := &cobra.Command{
		Use:   "run [name]",
		Short: "Run an action on its bound hosts",
		Args:  cobra.ExactArgs(1),
		RunE:  runAction,
	}
	runCmd.Flags().StringSlice("set", nil, "Set variable values (e.g., --set container=sigma_merl)")
	runCmd.Flags().String("host-alias", "", "Run on specific host (must be bound to action)")
	runCmd.Flags().String("alias", "", "Alias of bound host to run on")

	cmd.AddCommand(listCmd, infoCmd, addCmd, editCmd, deleteCmd, runCmd)

	return cmd
}

func SetDB(db *storage.DB) {
	actionsDB = db
}

func listActions(cmd *cobra.Command, _ []string) error {
	if actionsDB == nil {
		return fmt.Errorf("database not initialized")
	}

	actions, err := actionsDB.ListActions()
	if err != nil {
		return fmt.Errorf("failed to list actions: %w", err)
	}

	if len(actions) == 0 {
		fmt.Println(style.Dim.Render("No actions found. Run "), style.Info.Render("lissh actions add"), style.Dim.Render(" to create one."))
		return nil
	}

	fmt.Printf("  %-20s  %-40s  %s\n",
		style.Header.Render("Name"),
		style.Header.Render("Description"),
		style.Header.Render("Hosts"))
	fmt.Println(style.Subtle.Render("  ──────────────────────────────────────────────────────────────────────────────────────────────────────────────"))

	for _, a := range actions {
		desc := "(no description)"
		if a.Description != nil && *a.Description != "" {
			desc = *a.Description
		}
		hostAliases := []string{}
		for _, h := range a.Hosts {
			if h.Alias != nil && *h.Alias != "" {
				hostAliases = append(hostAliases, *h.Alias)
			} else {
				hostAliases = append(hostAliases, h.Hostname)
			}
		}
		hostsStr := strings.Join(hostAliases, ", ")
		if hostsStr == "" {
			hostsStr = "(unbound)"
		}

		fmt.Printf("  %-20s  %-40s  %s\n",
			style.Bold.Render(a.Name),
			truncate(desc, 40),
			style.Info.Render(truncate(hostsStr, 30)))
	}

	return nil
}

func showActionInfo(cmd *cobra.Command, args []string) error {
	if actionsDB == nil {
		return fmt.Errorf("database not initialized")
	}

	name := args[0]
	action, err := actionsDB.GetActionByName(name)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("action not found: %s", name)
	}

	fmt.Println()
	fmt.Printf("  %s %s\n", style.Info.Render("Name:"), style.Bold.Render(action.Name))
	if action.Description != nil && *action.Description != "" {
		fmt.Printf("  %s %s\n", style.Info.Render("Description:"), *action.Description)
	}
	fmt.Printf("  %s\n", style.Info.Render("Command:"))
	fmt.Println(style.Subtle.Render("    " + action.Command))

	vars := extractVariables(action.Command)
	if len(vars) > 0 {
		fmt.Printf("\n  %s %s\n", style.Warning.Render("Variables:"), style.Bold.Render(strings.Join(vars, ", ")))
		fmt.Println(style.Dim.Render("    Use --set to provide values when running"))
	}

	if len(action.Hosts) > 0 {
		fmt.Printf("\n  %s\n", style.Info.Render("Bound Hosts:"))
		for _, h := range action.Hosts {
			alias := h.Hostname
			if h.Alias != nil && *h.Alias != "" {
				alias = *h.Alias
			}
			fmt.Printf("    - %s (%s)\n", style.OK.Render(alias), h.Hostname)
		}
	} else {
		fmt.Printf("\n  %s\n", style.Warning.Render("No hosts bound (use --host-alias when running)"))
	}

	fmt.Printf("\n  %s %s\n", style.Info.Render("Created:"), style.Subtle.Render(action.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  %s %s\n", style.Info.Render("Updated:"), style.Subtle.Render(action.UpdatedAt.Format("2006-01-02 15:04:05")))

	return nil
}

func addAction(cmd *cobra.Command, args []string) error {
	if actionsDB == nil {
		return fmt.Errorf("database not initialized")
	}

	name := args[0]
	description, _ := cmd.Flags().GetString("description")
	command, _ := cmd.Flags().GetString("command")
	hostAliasesStr, _ := cmd.Flags().GetString("host-alias")

	if command == "" {
		return fmt.Errorf("--command is required")
	}

	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	hostAliases := []string{}
	if hostAliasesStr != "" {
		for _, alias := range strings.Split(hostAliasesStr, ",") {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				hostAliases = append(hostAliases, alias)
			}
		}
	}

	input := storage.CreateActionInput{
		Name:        name,
		Description: descPtr,
		Command:     command,
		HostAliases: hostAliases,
	}

	action, err := actionsDB.CreateAction(input)
	if err != nil {
		return fmt.Errorf("failed to create action: %w", err)
	}

	fmt.Printf("  %s Action %s created successfully\n", style.OK.Render("✓"), style.Bold.Render(action.Name))
	if len(action.Hosts) > 0 {
		fmt.Printf("  %s Bound to %d host(s)\n", style.Info.Render("→"), len(action.Hosts))
	}

	return nil
}

func editAction(cmd *cobra.Command, args []string) error {
	if actionsDB == nil {
		return fmt.Errorf("database not initialized")
	}

	name := args[0]
	action, err := actionsDB.GetActionByName(name)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("action not found: %s", name)
	}

	var descPtr *string
	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		descPtr = &desc
	}

	var cmdPtr *string
	if cmd.Flags().Changed("command") {
		c, _ := cmd.Flags().GetString("command")
		cmdPtr = &c
	}

	var hostAliases *[]string
	if cmd.Flags().Changed("host-alias") {
		hostAliasesStr, _ := cmd.Flags().GetString("host-alias")
		aliases := []string{}
		for _, alias := range strings.Split(hostAliasesStr, ",") {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				aliases = append(aliases, alias)
			}
		}
		hostAliases = &aliases
	}

	input := storage.UpdateActionInput{
		Name:        nil,
		Description: descPtr,
		Command:     cmdPtr,
		HostAliases: hostAliases,
	}

	if err := actionsDB.UpdateAction(action.ID, input); err != nil {
		return fmt.Errorf("failed to update action: %w", err)
	}

	fmt.Printf("  %s Action %s updated\n", style.OK.Render("✓"), style.Bold.Render(name))
	return nil
}

func deleteAction(cmd *cobra.Command, args []string) error {
	if actionsDB == nil {
		return fmt.Errorf("database not initialized")
	}

	name := args[0]
	action, err := actionsDB.GetActionByName(name)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("action not found: %s", name)
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		fmt.Printf("Delete action %s? (y/N): ", style.Bold.Render(name))
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	if err := actionsDB.DeleteAction(action.ID); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}

	fmt.Printf("  %s Action %s deleted\n", style.OK.Render("✓"), style.Bold.Render(name))
	return nil
}

func runAction(cmd *cobra.Command, args []string) error {
	if actionsDB == nil {
		return fmt.Errorf("database not initialized")
	}

	name := args[0]
	setVars, _ := cmd.Flags().GetStringSlice("set")
	overrideHost, _ := cmd.Flags().GetString("host-alias")
	alias, _ := cmd.Flags().GetString("alias")

	action, err := actionsDB.GetActionByName(name)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("action not found: %s", name)
	}

	vars := extractVariables(action.Command)

	varSet := parseSetVars(setVars)

	missingVars := []string{}
	for _, v := range vars {
		if _, ok := varSet[v]; !ok {
			missingVars = append(missingVars, v)
		}
	}

	if len(missingVars) > 0 {
		fmt.Printf("  %s The following variables need values:\n", style.Warning.Render("!"))
		for _, v := range missingVars {
			fmt.Printf("    %s ", style.Bold.Render(v))
			fmt.Printf("(empty to skip): ")
			var value string
			if _, err := fmt.Scanln(&value); err != nil {
				value = ""
			}
			if value != "" {
				varSet[v] = value
			}
		}
	}

	finalCommand := action.Command
	for v, val := range varSet {
		finalCommand = strings.ReplaceAll(finalCommand, "${"+v+"}", val)
	}

	var hosts []*storage.Host

	switch {
	case alias != "":
		if len(action.Hosts) == 0 {
			return fmt.Errorf("action %s has no bound hosts", name)
		}
		found := false
		for _, h := range action.Hosts {
			hostAlias := h.Hostname
			if h.Alias != nil && *h.Alias != "" {
				hostAlias = *h.Alias
			}
			if hostAlias == alias {
				hosts = []*storage.Host{h}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("host %s is not bound to action %s", alias, name)
		}
	case overrideHost != "":
		host, err := actionsDB.GetHostByHostname(overrideHost)
		if err != nil {
			return fmt.Errorf("failed to find host: %w", err)
		}
		if host == nil {
			return fmt.Errorf("host not found: %s", overrideHost)
		}
		hosts = []*storage.Host{host}
	default:
		hosts = action.Hosts
	}

	if len(hosts) == 0 {
		return fmt.Errorf("no hosts bound to this action (use --host-alias to override)")
	}

	for i, host := range hosts {
		if len(hosts) > 1 {
			fmt.Printf("\n%s [%d/%d] %s %s\n", style.Header.Render("→"), i+1, len(hosts), style.Bold.Render("Host:"), host.Hostname)
			if host.Alias != nil && *host.Alias != "" {
				fmt.Printf("  %s %s\n", style.Info.Render("Alias:"), *host.Alias)
			}
			fmt.Println(style.Subtle.Render("  ───────────────────────────────────────────────────────────────────────"))
		}

		if err := executeOnHost(host, finalCommand); err != nil {
			fmt.Fprintf(os.Stderr, "  %s %s\n", style.Error.Render("Error:"), err)
		}
	}

	return nil
}

func executeOnHost(host *storage.Host, command string) error {
	target := host.Hostname
	if host.IPAddress != nil && *host.IPAddress != "" {
		target = *host.IPAddress
	}

	user := ""
	if host.User != nil && *host.User != "" {
		user = *host.User
	}

	sshArgs := []string{}
	if user != "" {
		sshArgs = append(sshArgs, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null")
		sshArgs = append(sshArgs, user+"@"+target)
	} else {
		sshArgs = append(sshArgs, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null")
		sshArgs = append(sshArgs, target)
	}

	if host.Port != 0 && host.Port != 22 {
		sshArgs = append(sshArgs, "-p", strconv.Itoa(host.Port))
	}

	sshArgs = append(sshArgs, command)

	sshCmd := exec.Command("ssh", sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func extractVariables(command string) []string {
	matches := variableRegex.FindAllStringSubmatch(command, -1)
	seen := make(map[string]bool)
	var vars []string

	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			vars = append(vars, m[1])
		}
	}

	sort.Strings(vars)
	return vars
}

func parseSetVars(setVars []string) map[string]string {
	result := make(map[string]string)
	for _, sv := range setVars {
		parts := strings.SplitN(sv, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-2] + ".."
	}
	return s
}
