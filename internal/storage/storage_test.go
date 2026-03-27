package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	if db.path != dbPath {
		t.Errorf("expected path %s, got %s", dbPath, db.path)
	}
}

func TestCreateAndGetHost(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	alias := "test-server"
	input := CreateHostInput{
		Hostname: "192.168.1.100",
		Alias:    &alias,
		Port:     22,
		Source:   "test",
	}

	host, err := db.CreateHost(input)
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	if host.ID == 0 {
		t.Error("expected non-zero host ID")
	}
	if host.Hostname != "192.168.1.100" {
		t.Errorf("expected hostname 192.168.1.100, got %s", host.Hostname)
	}
	if host.Alias == nil || *host.Alias != "test-server" {
		t.Error("expected alias test-server")
	}

	retrieved, err := db.GetHostByID(host.ID)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected to retrieve host")
	}
	if retrieved.Hostname != "192.168.1.100" {
		t.Errorf("expected hostname 192.168.1.100, got %s", retrieved.Hostname)
	}
}

func TestListHosts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	for i := 1; i <= 3; i++ {
		_, err := db.CreateHost(CreateHostInput{
			Hostname: "host" + string(rune('0'+i)),
			Port:     22,
			Source:   "test",
		})
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}
	}

	hosts, err := db.ListHosts(true)
	if err != nil {
		t.Fatalf("failed to list hosts: %v", err)
	}
	if len(hosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(hosts))
	}
}

func TestHostExists(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	exists, err := db.HostExists("nonexistent")
	if err != nil {
		t.Fatalf("failed to check host existence: %v", err)
	}
	if exists {
		t.Error("expected host to not exist")
	}

	_, err = db.CreateHost(CreateHostInput{
		Hostname: "test-host",
		Port:     22,
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	exists, err = db.HostExists("test-host")
	if err != nil {
		t.Fatalf("failed to check host existence: %v", err)
	}
	if !exists {
		t.Error("expected host to exist")
	}
}

func TestSearchHosts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	alias := "web-server"
	_, err = db.CreateHost(CreateHostInput{
		Hostname: "192.168.1.10",
		Alias:    &alias,
		Port:     22,
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	_, err = db.CreateHost(CreateHostInput{
		Hostname: "192.168.1.20",
		Port:     22,
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	results, err := db.SearchHosts("web")
	if err != nil {
		t.Fatalf("failed to search hosts: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestUpdateHost(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	host, err := db.CreateHost(CreateHostInput{
		Hostname: "192.168.1.100",
		Port:     22,
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	newAlias := "updated-alias"
	err = db.UpdateHost(host.ID, &newAlias, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to update host: %v", err)
	}

	updated, err := db.GetHostByID(host.ID)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}
	if updated.Alias == nil || *updated.Alias != "updated-alias" {
		t.Error("expected alias to be updated")
	}
}

func TestMarkHostInactive(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	host, err := db.CreateHost(CreateHostInput{
		Hostname: "192.168.1.100",
		Port:     22,
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	err = db.MarkHostInactive(host.ID)
	if err != nil {
		t.Fatalf("failed to mark host inactive: %v", err)
	}

	hosts, err := db.ListHosts(false)
	if err != nil {
		t.Fatalf("failed to list hosts: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 active hosts, got %d", len(hosts))
	}

	hosts, err = db.ListHosts(true)
	if err != nil {
		t.Fatalf("failed to list all hosts: %v", err)
	}
	if len(hosts) != 1 {
		t.Errorf("expected 1 total host, got %d", len(hosts))
	}
}

func TestCreateSSHKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	fingerprint := "SHA256:abc123..."
	key, err := db.CreateSSHKey(CreateSSHKeyInput{
		Name:        "test-key",
		Path:        "/home/user/.ssh/id_rsa",
		KeyType:     "rsa",
		SizeBits:    intPtr(4096),
		Fingerprint: &fingerprint,
	})
	if err != nil {
		t.Fatalf("failed to create SSH key: %v", err)
	}

	if key.ID == 0 {
		t.Error("expected non-zero key ID")
	}
	if key.Name != "test-key" {
		t.Errorf("expected key name test-key, got %s", key.Name)
	}
}

func TestHistory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	host, err := db.CreateHost(CreateHostInput{
		Hostname: "192.168.1.100",
		Port:     22,
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	session, err := db.StartSession(host.ID)
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	if session.HostID != host.ID {
		t.Errorf("expected host ID %d, got %d", host.ID, session.HostID)
	}

	err = db.EndSession(session.ID)
	if err != nil {
		t.Fatalf("failed to end session: %v", err)
	}

	history, err := db.ListHistory(10, 0)
	if err != nil {
		t.Fatalf("failed to list history: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}
}

func intPtr(i int) *int {
	return &i
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
