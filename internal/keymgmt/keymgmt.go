package keymgmt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type KeyMetadata struct {
	SizeBits     *int
	KeyType      string
	Fingerprint  string
	Comment      *string
	LastModified string
}

func GetKeyMetadata(path string) (*KeyMetadata, error) {
	var metadata KeyMetadata

	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat key file: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		metadata.KeyType = "rsa"
		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			size := key.Size() * 8
			metadata.SizeBits = &size
		}
	case "OPENSSH PRIVATE KEY":
		metadata.KeyType = "ed25519"
		size := 256
		metadata.SizeBits = &size
	default:
		metadata.KeyType = block.Type
	}

	fp, err := getFingerprint(path)
	if err == nil {
		metadata.Fingerprint = fp
	}

	if pubPath := path + ".pub"; true {
		if fi, err := os.Stat(pubPath); err == nil {
			if data, err := os.ReadFile(pubPath); err == nil {
				parts := strings.Fields(string(data))
				if len(parts) >= 3 {
					metadata.Comment = &parts[2]
				}
			}
			metadata.LastModified = fi.ModTime().Format("2006-01-02 15:04:05")
		}
	}

	return &metadata, nil
}

func getFingerprint(path string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-lf", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}

	return "", fmt.Errorf("could not parse fingerprint")
}

func ParseKeyType(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return "", fmt.Errorf("not a valid PEM file")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		if _, err := x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
			return "", err
		}
		return "rsa", nil
	case "OPENSSH PRIVATE KEY":
		return "ed25519", nil
	case "EC PRIVATE KEY":
		return "ecdsa", nil
	default:
		return strings.ToLower(block.Type), nil
	}
}

func GetKeySize(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return 0, fmt.Errorf("not a valid PEM file")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return 0, err
		}
		return key.Size() * 8, nil
	case "OPENSSH PRIVATE KEY":
		return 256, nil
	default:
		return 0, fmt.Errorf("unknown key type: %s", block.Type)
	}
}

func ValidateKeyPair(pubPath, privPath string) (bool, error) {
	cmd := exec.Command("ssh-keygen", "-yf", privPath)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	pubData, err := os.ReadFile(pubPath)
	if err != nil {
		return false, err
	}

	pubKey := strings.TrimSpace(string(pubData))
	computedKey := strings.TrimSpace(string(output))

	parts := strings.Fields(pubKey)
	compParts := strings.Fields(computedKey)
	if len(parts) < 2 || len(compParts) < 2 {
		return false, fmt.Errorf("invalid key format")
	}

	return parts[1] == compParts[1], nil
}

func GenerateFingerprint(keyType string, keyData []byte) (string, error) {
	switch keyType {
	case "rsa":
		key, err := x509.ParsePKCS1PrivateKey(keyData)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %d %s", "rsa", key.Size()*8, "generated"), nil
	case "ed25519":
		if len(keyData) < 32 {
			return "", fmt.Errorf("invalid ed25519 key data")
		}
		return fmt.Sprintf("%s %d %s", "ed25519", 256, "generated"), nil
	default:
		return "", fmt.Errorf("unsupported key type: %s", keyType)
	}
}

func SafeDeleteKey(path string, dryRun bool) error {
	if dryRun {
		fmt.Printf("Would delete: %s\n", path)
		fmt.Printf("Would delete: %s.pub\n", path)
		return nil
	}

	paths := []string{path, path + ".pub"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("failed to delete %s: %w", p, err)
			}
		}
	}

	return nil
}

func EnsureKeyPermissions(path string) error {
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	return nil
}

func CreateSSHKey(keyType string, bits int, comment string) (string, string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current user: %w", err)
	}
	keyPath := filepath.Join(usr.HomeDir, ".ssh", fmt.Sprintf("id_%s_%d", keyType, bits))

	cmd := exec.Command("ssh-keygen", "-t", keyType, "-b", strconv.Itoa(bits), "-f", keyPath, "-C", comment, "-N", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	return keyPath, keyPath + ".pub", nil
}

func CreatePublicKeyFromPrivate(pubKey interface{}, keyType, comment string) ([]byte, error) {
	switch keyType {
	case "rsa":
		cmd := exec.Command("ssh-keygen", "-y", "-t", "rsa", "-f", "/dev/null")
		cmd.Stdin, _ = os.Open(os.DevNull)
		return cmd.Output()
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

var _ = rsa.PrivateKey{}
