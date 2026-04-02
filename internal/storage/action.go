package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Action struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Command     string    `json:"command"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Hosts       []*Host   `json:"hosts,omitempty"`
}

type ActionHost struct {
	ActionID int64 `json:"action_id"`
	HostID   int64 `json:"host_id"`
}

type CreateActionInput struct {
	Name        string
	Description *string
	Command     string
	HostAliases []string
}

type UpdateActionInput struct {
	Name        *string
	Description *string
	Command     *string
	HostAliases *[]string
}

func (db *DB) CreateAction(input CreateActionInput) (*Action, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.Exec(`
		INSERT INTO actions (name, description, command)
		VALUES (?, ?, ?)
	`, input.Name, input.Description, input.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	for _, alias := range input.HostAliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		host, err := db.GetHostByHostname(alias)
		if err != nil {
			return nil, fmt.Errorf("failed to find host %s: %w", alias, err)
		}
		if host == nil {
			return nil, fmt.Errorf("host not found: %s", alias)
		}
		_, err = tx.Exec(`
			INSERT INTO action_hosts (action_id, host_id) VALUES (?, ?)
		`, id, host.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to link host to action: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return db.GetActionByID(id)
}

func (db *DB) GetActionByID(id int64) (*Action, error) {
	action := &Action{}
	var description sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, description, command, created_at, updated_at
		FROM actions WHERE id = ?
	`, id).Scan(&action.ID, &action.Name, &description, &action.Command, &action.CreatedAt, &action.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	if description.Valid {
		action.Description = &description.String
	}

	hosts, err := db.GetHostsForAction(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action hosts: %w", err)
	}
	action.Hosts = hosts

	return action, nil
}

func (db *DB) GetActionByName(name string) (*Action, error) {
	action := &Action{}
	var description sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, description, command, created_at, updated_at
		FROM actions WHERE name = ?
	`, name).Scan(&action.ID, &action.Name, &description, &action.Command, &action.CreatedAt, &action.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	if description.Valid {
		action.Description = &description.String
	}

	hosts, err := db.GetHostsForAction(action.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get action hosts: %w", err)
	}
	action.Hosts = hosts

	return action, nil
}

func (db *DB) ListActions() ([]*Action, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, description, command, created_at, updated_at
		FROM actions ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	defer rows.Close()

	var actions []*Action
	for rows.Next() {
		action := &Action{}
		var description sql.NullString

		err := rows.Scan(&action.ID, &action.Name, &description, &action.Command, &action.CreatedAt, &action.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}

		if description.Valid {
			action.Description = &description.String
		}

		hosts, err := db.GetHostsForAction(action.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get action hosts: %w", err)
		}
		action.Hosts = hosts

		actions = append(actions, action)
	}

	return actions, nil
}

func (db *DB) GetHostsForAction(actionID int64) ([]*Host, error) {
	rows, err := db.conn.Query(`
		SELECT h.id, h.hostname, h.alias, h.ip_address, h.user, h.port, h.source, h.ssh_key_id, h.notes, h.is_inactive, h.discovered_at, h.created_at, h.updated_at
		FROM hosts h
		INNER JOIN action_hosts ah ON h.id = ah.host_id
		WHERE ah.action_id = ?
		ORDER BY h.hostname
	`, actionID)
	if err != nil {
		return nil, err
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

func (db *DB) UpdateAction(id int64, input UpdateActionInput) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `UPDATE actions SET updated_at = CURRENT_TIMESTAMP`
	args := []interface{}{}

	if input.Name != nil {
		query += `, name = ?`
		args = append(args, *input.Name)
	}
	if input.Description != nil {
		query += `, description = ?`
		args = append(args, *input.Description)
	}
	if input.Command != nil {
		query += `, command = ?`
		args = append(args, *input.Command)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update action: %w", err)
	}

	if input.HostAliases != nil {
		_, err = tx.Exec(`DELETE FROM action_hosts WHERE action_id = ?`, id)
		if err != nil {
			return fmt.Errorf("failed to clear action hosts: %w", err)
		}

		for _, alias := range *input.HostAliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			host, err := db.GetHostByHostname(alias)
			if err != nil {
				return fmt.Errorf("failed to find host %s: %w", alias, err)
			}
			if host == nil {
				return fmt.Errorf("host not found: %s", alias)
			}
			_, err = tx.Exec(`
				INSERT INTO action_hosts (action_id, host_id) VALUES (?, ?)
			`, id, host.ID)
			if err != nil {
				return fmt.Errorf("failed to link host to action: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *DB) DeleteAction(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM action_hosts WHERE action_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete action hosts: %w", err)
	}

	_, err = db.conn.Exec(`DELETE FROM actions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}

	return nil
}

func (db *DB) AddMigrationActionHosts() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS action_hosts (
			action_id INTEGER NOT NULL,
			host_id INTEGER NOT NULL,
			PRIMARY KEY (action_id, host_id),
			FOREIGN KEY (action_id) REFERENCES actions(id) ON DELETE CASCADE,
			FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE CASCADE
		)
	`)
	return err
}
