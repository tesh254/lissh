package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type Host struct {
	ID           int64     `json:"id"`
	Hostname     string    `json:"hostname"`
	Alias        *string   `json:"alias,omitempty"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	User         *string   `json:"user,omitempty"`
	Port         int       `json:"port"`
	Source       string    `json:"source"`
	SSHKeyID     *int64    `json:"ssh_key_id,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	IsInactive   bool      `json:"is_inactive"`
	DiscoveredAt time.Time `json:"discovered_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateHostInput struct {
	Hostname  string
	Alias     *string
	IPAddress *string
	User      *string
	Port      int
	Source    string
	SSHKeyID  *int64
	Notes     *string
}

func (db *DB) CreateHost(input CreateHostInput) (*Host, error) {
	result, err := db.conn.Exec(`
		INSERT INTO hosts (hostname, alias, ip_address, user, port, source, ssh_key_id, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, input.Hostname, input.Alias, input.IPAddress, input.User, input.Port, input.Source, input.SSHKeyID, input.Notes)
	if err != nil {
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return db.GetHostByID(id)
}

func (db *DB) GetHostByID(id int64) (*Host, error) {
	host := &Host{}
	var alias, ip, notes, user sql.NullString
	var sshKeyID sql.NullInt64

	err := db.conn.QueryRow(`
		SELECT id, hostname, alias, ip_address, user, port, source, ssh_key_id, notes, is_inactive, discovered_at, created_at, updated_at
		FROM hosts WHERE id = ?
	`, id).Scan(
		&host.ID, &host.Hostname, &alias, &ip, &user, &host.Port, &host.Source,
		&sshKeyID, &notes, &host.IsInactive, &host.DiscoveredAt, &host.CreatedAt, &host.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	if alias.Valid {
		host.Alias = &alias.String
	}
	if ip.Valid {
		host.IPAddress = &ip.String
	}
	if user.Valid {
		host.User = &user.String
	}
	if sshKeyID.Valid {
		host.SSHKeyID = &sshKeyID.Int64
	}
	if notes.Valid {
		host.Notes = &notes.String
	}

	return host, nil
}

func (db *DB) GetHostByHostname(hostname string) (*Host, error) {
	host := &Host{}
	var alias, ip, notes, user sql.NullString
	var sshKeyID sql.NullInt64

	err := db.conn.QueryRow(`
		SELECT id, hostname, alias, ip_address, user, port, source, ssh_key_id, notes, is_inactive, discovered_at, created_at, updated_at
		FROM hosts WHERE hostname = ? OR alias = ?
	`, hostname, hostname).Scan(
		&host.ID, &host.Hostname, &alias, &ip, &user, &host.Port, &host.Source,
		&sshKeyID, &notes, &host.IsInactive, &host.DiscoveredAt, &host.CreatedAt, &host.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	if alias.Valid {
		host.Alias = &alias.String
	}
	if ip.Valid {
		host.IPAddress = &ip.String
	}
	if user.Valid {
		host.User = &user.String
	}
	if sshKeyID.Valid {
		host.SSHKeyID = &sshKeyID.Int64
	}
	if notes.Valid {
		host.Notes = &notes.String
	}

	return host, nil
}

func (db *DB) ListHosts(includeInactive bool) ([]*Host, error) {
	query := `SELECT id, hostname, alias, ip_address, user, port, source, ssh_key_id, notes, is_inactive, discovered_at, created_at, updated_at FROM hosts`
	if !includeInactive {
		query += ` WHERE is_inactive = 0`
	}
	query += ` ORDER BY COALESCE(alias, hostname)`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}
	defer rows.Close()

	var hosts []*Host
	for rows.Next() {
		host := &Host{}
		var alias, ip, notes, user sql.NullString
		var sshKeyID sql.NullInt64

		err := rows.Scan(
			&host.ID, &host.Hostname, &alias, &ip, &user, &host.Port, &host.Source,
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
		if user.Valid {
			host.User = &user.String
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

func (db *DB) SearchHosts(term string) ([]*Host, error) {
	query := `
		SELECT id, hostname, alias, ip_address, user, port, source, ssh_key_id, notes, is_inactive, discovered_at, created_at, updated_at 
		FROM hosts 
		WHERE is_inactive = 0 AND (hostname LIKE ? OR alias LIKE ? OR ip_address LIKE ? OR notes LIKE ? OR user LIKE ?)
		ORDER BY COALESCE(alias, hostname)
	`
	pattern := "%" + term + "%"

	rows, err := db.conn.Query(query, pattern, pattern, pattern, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search hosts: %w", err)
	}
	defer rows.Close()

	var hosts []*Host
	for rows.Next() {
		host := &Host{}
		var alias, ip, notes, user sql.NullString
		var sshKeyID sql.NullInt64

		err := rows.Scan(
			&host.ID, &host.Hostname, &alias, &ip, &user, &host.Port, &host.Source,
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
		if user.Valid {
			host.User = &user.String
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

func (db *DB) UpdateHost(id int64, alias *string, notes *string, sshKeyID *int64, user *string) error {
	query := `UPDATE hosts SET updated_at = CURRENT_TIMESTAMP`
	args := []interface{}{}

	if alias != nil {
		query += `, alias = ?`
		args = append(args, *alias)
	}
	if notes != nil {
		query += `, notes = ?`
		args = append(args, *notes)
	}
	if sshKeyID != nil {
		query += `, ssh_key_id = ?`
		args = append(args, *sshKeyID)
	}
	if user != nil {
		query += `, user = ?`
		args = append(args, *user)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update host: %w", err)
	}

	return nil
}

func (db *DB) MarkHostInactive(id int64) error {
	_, err := db.conn.Exec(`UPDATE hosts SET is_inactive = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to mark host inactive: %w", err)
	}
	return nil
}

func (db *DB) DeleteHost(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM hosts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete host: %w", err)
	}
	return nil
}

func (db *DB) HostExists(hostname string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM hosts WHERE hostname = ?`, hostname).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check host existence: %w", err)
	}
	return count > 0, nil
}

func (db *DB) UpdateHostPort(id int64, port int) error {
	_, err := db.conn.Exec(`UPDATE hosts SET port = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, port, id)
	if err != nil {
		return fmt.Errorf("failed to update host port: %w", err)
	}
	return nil
}

func (db *DB) BulkMarkInactive(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query := `UPDATE hosts SET is_inactive = 1, updated_at = CURRENT_TIMESTAMP WHERE id IN (`
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += `,`
		}
		query += `?`
		args[i] = id
	}
	query += `)`

	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk mark inactive: %w", err)
	}
	return nil
}
