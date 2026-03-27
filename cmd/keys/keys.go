package keys

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wcrg/lissh/internal/keymgmt"
	"github.com/wcrg/lissh/internal/storage"
	"github.com/wcrg/lissh/pkg/style"
)

var keysDB *storage.DB

func NewKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage SSH keys",
		Long:  `Create, list, and manage SSH keys for authentication.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all SSH keys",
		RunE:  listKeys,
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan ~/.ssh for keys and add them to lissh",
		RunE:  scanKeys,
	}

	infoCmd := &cobra.Command{
		Use:   "info [id]",
		Short: "Show detailed info about a key",
		Args:  cobra.ExactArgs(1),
		RunE:  keyInfo,
	}

	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new SSH keypair",
		Args:  cobra.ExactArgs(1),
		RunE:  createKey,
	}
	createCmd.Flags().String("type", "ed25519", "Key type (rsa, ed25519)")
	createCmd.Flags().Int("bits", 0, "Key size in bits (default: 4096 for RSA, 256 for Ed25519)")
	createCmd.Flags().String("comment", "", "Comment for the key")

	associateCmd := &cobra.Command{
		Use:   "associate [key-id] [host-id]",
		Short: "Associate a key with a host",
		Args:  cobra.ExactArgs(2),
		RunE:  associateKey,
	}

	removeCmd := &cobra.Command{
		Use:   "remove [id]",
		Short: "Remove an SSH key from lissh (not from filesystem)",
		Args:  cobra.ExactArgs(1),
		RunE:  removeKey,
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Permanently delete an SSH key and its associations",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteKey,
	}
	deleteCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")
	deleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")

	cmd.AddCommand(listCmd, scanCmd, infoCmd, createCmd, associateCmd, removeCmd, deleteCmd)

	return cmd
}

func SetDB(db *storage.DB) {
	keysDB = db
}

func listKeys(cmd *cobra.Command, _ []string) error {
	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}
	keys, err := keysDB.ListSSHKeys()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println(style.Dim.Render("No SSH keys registered. Use "), style.Info.Render("lissh keys scan"), style.Dim.Render(" to scan ~/.ssh or "), style.Info.Render("lissh keys create"), style.Dim.Render(" to create a new one."))
		return nil
	}

	fmt.Printf("\n  %-4s  %-22s  %-10s  %-8s  %-40s  %s\n",
		style.Header.Render("ID"),
		style.Header.Render("Name"),
		style.Header.Render("Type"),
		style.Header.Render("Bits"),
		style.Header.Render("Private"),
		style.Header.Render("Public"))
	fmt.Println(style.Subtle.Render("  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────"))
	for _, k := range keys {
		size := style.Dim.Render("N/A")
		if k.SizeBits != nil {
			size = fmt.Sprintf("%d", *k.SizeBits)
		}
		keyType := style.Bold.Render(k.KeyType)
		if k.KeyType == "ed25519" {
			keyType = style.OK.Render("ed25519")
		} else if k.KeyType == "rsa" {
			keyType = style.Warning.Render("rsa")
		}
		pubPath := style.Dim.Render("N/A")
		if k.PublicKeyPath != nil {
			pubPath = style.Dim.Render(truncateKey(*k.PublicKeyPath, 40))
		}
		fmt.Printf("  %-4d  %-22s  %-10s  %-8s  %-40s  %s\n",
			k.ID,
			truncateKey(k.Name, 22),
			keyType,
			size,
			style.Dim.Render(truncateKey(k.Path, 40)),
			pubPath,
		)
	}
	fmt.Println()

	return nil
}

func scanKeys(cmd *cobra.Command, _ []string) error {
	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}

	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	sshDir := filepath.Join(usr.HomeDir, ".ssh")

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return fmt.Errorf("failed to read ~/.ssh: %w", err)
	}

	var newKeys []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "known_hosts" || name == "config" || name == "authorized_keys" || name == "config.lock" {
			continue
		}
		if name == "id_rsa" || name == "id_ed25519" || name == "id_ecdsa" || name == "id_dsa" {
			newKeys = append(newKeys, filepath.Join(sshDir, name))
		}
		if strings.HasSuffix(name, "_rsa") || strings.HasSuffix(name, "_ed25519") || strings.HasSuffix(name, "_ecdsa") || strings.HasSuffix(name, "_dsa") {
			newKeys = append(newKeys, filepath.Join(sshDir, name))
		}
	}

	if len(newKeys) == 0 {
		fmt.Println(style.Dim.Render("No new keys found in ~/.ssh"))
		return nil
	}

	fmt.Printf("\n  %s Found %d key(s) in ~/.ssh:\n\n", style.Header.Render("Scanning..."), len(newKeys))

	added := 0
	for _, keyPath := range newKeys {
		exists, _ := keysDB.GetSSHKeyByPath(keyPath)
		if exists != nil {
			fmt.Printf("  %s %s (already registered)\n", style.Inactive.Render("[=]"), style.Dim.Render(keyPath))
			continue
		}

		meta, err := keymgmt.GetKeyMetadata(keyPath)
		if err != nil {
			fmt.Printf("  %s %s (error reading: %v)\n", style.Warning.Render("[!]"), keyPath, err)
			continue
		}

		name := filepath.Base(keyPath)
		comment := ""
		if meta.Comment != nil {
			comment = *meta.Comment
		}

		publicKeyPath := keyPath + ".pub"
		_, err = keysDB.CreateSSHKey(storage.CreateSSHKeyInput{
			Name:          name,
			Path:          keyPath,
			PublicKeyPath: &publicKeyPath,
			KeyType:       meta.KeyType,
			SizeBits:      meta.SizeBits,
			Comment:       &comment,
			Fingerprint:   &meta.Fingerprint,
		})
		if err != nil {
			fmt.Printf("  %s %s (failed to add: %v)\n", style.Error.Render("[X]"), keyPath, err)
			continue
		}

		fmt.Printf("  %s %s (%s %d bits)\n", style.OK.Render("[+]"), style.Bold.Render(name), meta.KeyType, *meta.SizeBits)
		added++
	}

	fmt.Printf("\n  %s Added %d key(s)\n\n", style.Success.Render("Done!"), added)
	return nil
}

func truncateKey(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-2] + ".."
	}
	return s
}

func keyInfo(cmd *cobra.Command, args []string) error {
	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	key, err := keysDB.GetSSHKeyByID(id)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}
	if key == nil {
		return fmt.Errorf("key not found")
	}

	fmt.Println()
	fmt.Printf("  %s %d\n", style.Info.Render("ID:"), key.ID)
	fmt.Printf("  %s %s\n", style.Info.Render("Name:"), style.Bold.Render(key.Name))
	fmt.Printf("  %s %s\n", style.Info.Render("Private:"), style.Dim.Render(key.Path))
	if key.PublicKeyPath != nil {
		fmt.Printf("  %s %s\n", style.Info.Render("Public:"), style.Dim.Render(*key.PublicKeyPath))
	}
	fmt.Printf("  %s %s\n", style.Info.Render("Type:"), style.Bold.Render(key.KeyType))
	if key.SizeBits != nil {
		fmt.Printf("  %s %d bits\n", style.Info.Render("Size:"), *key.SizeBits)
	}
	if key.Comment != nil {
		fmt.Printf("  %s %s\n", style.Info.Render("Comment:"), *key.Comment)
	}
	if key.Fingerprint != nil {
		fmt.Printf("  %s %s\n", style.Info.Render("Fingerprint:"), style.Dim.Render(*key.Fingerprint))
	}
	fmt.Printf("  %s %s\n", style.Info.Render("Created:"), style.Dim.Render(key.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  %s %s\n", style.Info.Render("Modified:"), style.Dim.Render(key.UpdatedAt.Format("2006-01-02 15:04:05")))

	hosts, _ := keysDB.GetHostsBySSHKeyID(id)
	if len(hosts) > 0 {
		fmt.Println()
		fmt.Printf("  %s Associated hosts:\n", style.Info.Render(""))
		for _, h := range hosts {
			alias := h.Hostname
			if h.Alias != nil && *h.Alias != "" {
				alias = *h.Alias
			}
			fmt.Printf("    %s %s (%s)\n", style.OK.Render("[*]"), style.Bold.Render(alias), h.Hostname)
		}
	}
	fmt.Println()

	return nil
}

func createKey(cmd *cobra.Command, args []string) error {
	name := args[0]
	keyType, _ := cmd.Flags().GetString("type")
	bits, _ := cmd.Flags().GetInt("bits")
	comment, _ := cmd.Flags().GetString("comment")

	if bits == 0 {
		switch keyType {
		case "rsa":
			bits = 4096
		case "ed25519":
			bits = 256
		default:
			bits = 4096
		}
	}

	usr, _ := user.Current()
	keyDir := filepath.Join(usr.HomeDir, ".ssh")
	keyPath := filepath.Join(keyDir, name)

	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("key file already exists at %s", keyPath)
	}

	fmt.Printf("\n  %s Creating %s key with %d bits...\n", style.Header.Render("Creating"), keyType, bits)

	cmdExec := exec.Command("ssh-keygen", "-t", keyType, "-b", strconv.Itoa(bits), "-f", keyPath, "-C", comment, "-N", "")
	cmdExec.Stdin = os.Stdin
	cmdExec.Stdout = os.Stdout
	cmdExec.Stderr = os.Stderr
	if err := cmdExec.Run(); err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	meta, err := keymgmt.GetKeyMetadata(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read key metadata: %v\n", err)
	}

	var fp string
	if meta != nil {
		fp = meta.Fingerprint
	}

	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}
	publicKeyPath := keyPath + ".pub"
	_, err = keysDB.CreateSSHKey(storage.CreateSSHKeyInput{
		Name:          name,
		Path:          keyPath,
		PublicKeyPath: &publicKeyPath,
		KeyType:       keyType,
		SizeBits:      meta.SizeBits,
		Comment:       &comment,
		Fingerprint:   &fp,
	})
	if err != nil {
		return fmt.Errorf("failed to register key in database: %w", err)
	}

	fmt.Printf("\n  %s Key created successfully!\n", style.Success.Render("Done!"))
	fmt.Printf("\n  Private key: %s\n", style.Bold.Render(keyPath))
	fmt.Printf("  Public key:  %s.pub\n", keyPath)
	fmt.Printf("\n  %s Add to your servers with:\n", style.Info.Render("Remember:"))
	fmt.Printf("    cat %s.pub | ssh user@host 'cat >> ~/.ssh/authorized_keys'\n\n", keyPath)

	return nil
}

func associateKey(cmd *cobra.Command, args []string) error {
	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}
	keyIDStr := args[0]
	hostIDStr := args[1]

	keyID, err := strconv.ParseInt(keyIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}
	hostID, err := strconv.ParseInt(hostIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host ID: %w", err)
	}

	key, err := keysDB.GetSSHKeyByID(keyID)
	if err != nil || key == nil {
		return fmt.Errorf("key not found")
	}

	host, err := keysDB.GetHostByID(hostID)
	if err != nil || host == nil {
		return fmt.Errorf("host not found")
	}

	if err := keysDB.UpdateHost(hostID, nil, nil, &keyID, nil); err != nil {
		return fmt.Errorf("failed to associate key: %w", err)
	}

	fmt.Printf("\n  %s Associated key %s with host %s\n\n", style.Success.Render("Done!"), style.Bold.Render(key.Name), style.Bold.Render(host.Hostname))
	return nil
}

func removeKey(cmd *cobra.Command, args []string) error {
	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	key, err := keysDB.GetSSHKeyByID(id)
	if err != nil || key == nil {
		return fmt.Errorf("key not found")
	}

	if err := keysDB.DeleteSSHKey(id); err != nil {
		return fmt.Errorf("failed to remove key: %w", err)
	}

	fmt.Printf("\n  %s Key %s removed from lissh (file preserved at %s)\n\n", style.Success.Render("Done!"), style.Bold.Render(key.Name), style.Dim.Render(key.Path))
	return nil
}

func deleteKey(cmd *cobra.Command, args []string) error {
	if keysDB == nil {
		return fmt.Errorf("database not initialized")
	}
	idStr := args[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		key, _ := keysDB.GetSSHKeyByID(id)
		if key == nil {
			return fmt.Errorf("key not found")
		}
		fmt.Printf("\n  %s Would delete:\n", style.Warning.Render("Dry run"))
		fmt.Printf("    Key: %s\n", key.Name)
		fmt.Printf("    File: %s\n", key.Path)
		fmt.Printf("    Public: %s.pub\n\n", key.Path)
		return nil
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		key, _ := keysDB.GetSSHKeyByID(id)
		if key == nil {
			return fmt.Errorf("key not found")
		}
		fmt.Printf("\n  %s PERMANENTLY DELETE:\n", style.Error.Render("Warning!"))
		fmt.Printf("    Key record: %s\n", key.Name)
		fmt.Printf("    Private:   %s\n", key.Path)
		fmt.Printf("    Public:    %s.pub\n", key.Path)
		fmt.Print("\n  Continue? (y/N): ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		if response != "y" && response != "Y" {
			fmt.Println(style.Dim.Render("\n  Aborted\n"))
			return nil
		}
	}

	key, _ := keysDB.GetSSHKeyByID(id)
	if key != nil {
		os.Remove(key.Path)
		os.Remove(key.Path + ".pub")
	}

	if err := keysDB.DeleteSSHKey(id); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	fmt.Printf("\n  %s Key deleted successfully\n\n", style.Success.Render("Done!"))
	return nil
}
