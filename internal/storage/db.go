package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
	path string
}

func New(dbPath string) (*DB, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not find home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".lissh", "lissh.db")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("could not create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn, path: dbPath}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Path() string {
	return db.path
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS hosts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hostname TEXT NOT NULL UNIQUE,
			alias TEXT,
			ip_address TEXT,
			port INTEGER DEFAULT 22,
			source TEXT NOT NULL,
			ssh_key_id INTEGER,
			notes TEXT,
			is_inactive INTEGER DEFAULT 0,
			discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (ssh_key_id) REFERENCES ssh_keys(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_hosts_hostname ON hosts(hostname)`,
		`CREATE INDEX IF NOT EXISTS idx_hosts_alias ON hosts(alias)`,
		`CREATE INDEX IF NOT EXISTS idx_hosts_is_inactive ON hosts(is_inactive)`,
		`CREATE TABLE IF NOT EXISTS ssh_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			public_key_path TEXT,
			key_type TEXT NOT NULL,
			size_bits INTEGER,
			comment TEXT,
			fingerprint TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ssh_keys_path ON ssh_keys(path)`,
		`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER NOT NULL,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME,
			duration_seconds INTEGER,
			FOREIGN KEY (host_id) REFERENCES hosts(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_host_id ON history(host_id)`,
		`CREATE INDEX IF NOT EXISTS idx_history_started_at ON history(started_at)`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for i, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	if err := db.addUserColumnIfNotExists(); err != nil {
		return fmt.Errorf("failed to add user column: %w", err)
	}

	return nil
}

func (db *DB) addUserColumnIfNotExists() error {
	rows, err := db.conn.Query("PRAGMA table_info(hosts)")
	if err != nil {
		return err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var ci int
		var cn string
		var ct, df, notnull, pk interface{}
		if err := rows.Scan(&ci, &cn, &ct, &notnull, &df, &pk); err != nil {
			return err
		}
		columns = append(columns, cn)
	}

	for _, col := range columns {
		if col == "user" {
			return nil
		}
	}

	_, err = db.conn.Exec("ALTER TABLE hosts ADD COLUMN user TEXT")
	return err
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}
