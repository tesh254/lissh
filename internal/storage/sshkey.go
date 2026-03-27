package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type SSHKey struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	PublicKeyPath *string   `json:"public_key_path,omitempty"`
	KeyType       string    `json:"key_type"`
	SizeBits      *int      `json:"size_bits,omitempty"`
	Comment       *string   `json:"comment,omitempty"`
	Fingerprint   *string   `json:"fingerprint,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateSSHKeyInput struct {
	Name          string
	Path          string
	PublicKeyPath *string
	KeyType       string
	SizeBits      *int
	Comment       *string
	Fingerprint   *string
}

func (db *DB) CreateSSHKey(input CreateSSHKeyInput) (*SSHKey, error) {
	result, err := db.conn.Exec(`
		INSERT INTO ssh_keys (name, path, public_key_path, key_type, size_bits, comment, fingerprint)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, input.Name, input.Path, input.PublicKeyPath, input.KeyType, input.SizeBits, input.Comment, input.Fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh key: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return db.GetSSHKeyByID(id)
}

func (db *DB) GetSSHKeyByID(id int64) (*SSHKey, error) {
	key := &SSHKey{}
	var sizeBits sql.NullInt64
	var comment, fingerprint, publicKeyPath sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, path, public_key_path, key_type, size_bits, comment, fingerprint, created_at, updated_at
		FROM ssh_keys WHERE id = ?
	`, id).Scan(
		&key.ID, &key.Name, &key.Path, &publicKeyPath, &key.KeyType, &sizeBits,
		&comment, &fingerprint, &key.CreatedAt, &key.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ssh key: %w", err)
	}

	if sizeBits.Valid {
		size := int(sizeBits.Int64)
		key.SizeBits = &size
	}
	if comment.Valid {
		key.Comment = &comment.String
	}
	if fingerprint.Valid {
		key.Fingerprint = &fingerprint.String
	}
	if publicKeyPath.Valid {
		key.PublicKeyPath = &publicKeyPath.String
	}

	return key, nil
}

func (db *DB) GetSSHKeyByPath(path string) (*SSHKey, error) {
	key := &SSHKey{}
	var sizeBits sql.NullInt64
	var comment, fingerprint, publicKeyPath sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, path, public_key_path, key_type, size_bits, comment, fingerprint, created_at, updated_at
		FROM ssh_keys WHERE path = ?
	`, path).Scan(
		&key.ID, &key.Name, &key.Path, &publicKeyPath, &key.KeyType, &sizeBits,
		&comment, &fingerprint, &key.CreatedAt, &key.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ssh key: %w", err)
	}

	if sizeBits.Valid {
		size := int(sizeBits.Int64)
		key.SizeBits = &size
	}
	if comment.Valid {
		key.Comment = &comment.String
	}
	if fingerprint.Valid {
		key.Fingerprint = &fingerprint.String
	}
	if publicKeyPath.Valid {
		key.PublicKeyPath = &publicKeyPath.String
	}

	return key, nil
}

func (db *DB) ListSSHKeys() ([]*SSHKey, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, path, public_key_path, key_type, size_bits, comment, fingerprint, created_at, updated_at
		FROM ssh_keys ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list ssh keys: %w", err)
	}
	defer rows.Close()

	var keys []*SSHKey
	for rows.Next() {
		key := &SSHKey{}
		var sizeBits sql.NullInt64
		var comment, fingerprint, publicKeyPath sql.NullString

		err := rows.Scan(
			&key.ID, &key.Name, &key.Path, &publicKeyPath, &key.KeyType, &sizeBits,
			&comment, &fingerprint, &key.CreatedAt, &key.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ssh key: %w", err)
		}

		if sizeBits.Valid {
			size := int(sizeBits.Int64)
			key.SizeBits = &size
		}
		if comment.Valid {
			key.Comment = &comment.String
		}
		if fingerprint.Valid {
			key.Fingerprint = &fingerprint.String
		}
		if publicKeyPath.Valid {
			key.PublicKeyPath = &publicKeyPath.String
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (db *DB) UpdateSSHKey(id int64, name *string) error {
	query := `UPDATE ssh_keys SET updated_at = CURRENT_TIMESTAMP`
	args := []interface{}{}

	if name != nil {
		query += `, name = ?`
		args = append(args, *name)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update ssh key: %w", err)
	}

	return nil
}

func (db *DB) DeleteSSHKey(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM ssh_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete ssh key: %w", err)
	}
	return nil
}

func (db *DB) GetHostsBySSHKeyID(keyID int64) ([]*Host, error) {
	rows, err := db.conn.Query(`
		SELECT id, hostname, alias, ip_address, port, source, ssh_key_id, notes, is_inactive, discovered_at, created_at, updated_at
		FROM hosts WHERE ssh_key_id = ?
	`, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hosts by ssh key: %w", err)
	}
	defer rows.Close()

	var hosts []*Host
	for rows.Next() {
		host := &Host{}
		var alias, ip, notes sql.NullString
		var sshKeyID sql.NullInt64

		err := rows.Scan(
			&host.ID, &host.Hostname, &alias, &ip, &host.Port, &host.Source,
			&sshKeyID, &notes, &host.IsInactive, &host.DiscoveredAt, &host.CreatedAt, &host.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan host: %w", err)
		}

		if alias.Valid {
			host.Alias = &alias.String
		}
		if ip.Valid {
			host.IPAddress = &ip.String
		}
		if sshKeyID.Valid {
			host.SSHKeyID = &sshKeyID.Int64
		}
		if notes.Valid {
			host.Notes = &notes.String
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}
