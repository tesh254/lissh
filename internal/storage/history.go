package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type History struct {
	ID              int64      `json:"id"`
	HostID          int64      `json:"host_id"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	DurationSeconds *int64     `json:"duration_seconds,omitempty"`
}

type HistoryWithHost struct {
	History
	Hostname string  `json:"hostname"`
	Alias    *string `json:"alias,omitempty"`
}

func (db *DB) StartSession(hostID int64) (*History, error) {
	result, err := db.conn.Exec(`
		INSERT INTO history (host_id, started_at) VALUES (?, CURRENT_TIMESTAMP)
	`, hostID)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return db.GetHistoryByID(id)
}

func (db *DB) EndSession(id int64) error {
	result, err := db.conn.Exec(`
		UPDATE history 
		SET ended_at = CURRENT_TIMESTAMP, 
		    duration_seconds = CAST((julianday(CURRENT_TIMESTAMP) - julianday(started_at)) * 86400 AS INTEGER)
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("history entry not found")
	}

	return nil
}

func (db *DB) GetHistoryByID(id int64) (*History, error) {
	history := &History{}
	var endedAt sql.NullTime
	var durationSeconds sql.NullInt64

	err := db.conn.QueryRow(`
		SELECT id, host_id, started_at, ended_at, duration_seconds FROM history WHERE id = ?
	`, id).Scan(&history.ID, &history.HostID, &history.StartedAt, &endedAt, &durationSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	if endedAt.Valid {
		history.EndedAt = &endedAt.Time
	}
	if durationSeconds.Valid {
		history.DurationSeconds = &durationSeconds.Int64
	}

	return history, nil
}

func (db *DB) ListHistory(limit int, offset int) ([]*HistoryWithHost, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT h.id, h.host_id, h.started_at, h.ended_at, h.duration_seconds, 
		       hs.hostname, hs.alias
		FROM history h
		JOIN hosts hs ON h.host_id = hs.id
		ORDER BY h.started_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	var histories []*HistoryWithHost
	for rows.Next() {
		hw := &HistoryWithHost{}
		var endedAt sql.NullTime
		var durationSeconds sql.NullInt64
		var alias sql.NullString

		err := rows.Scan(
			&hw.ID, &hw.HostID, &hw.StartedAt, &endedAt, &durationSeconds,
			&hw.Hostname, &alias,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}

		if endedAt.Valid {
			hw.EndedAt = &endedAt.Time
		}
		if durationSeconds.Valid {
			hw.DurationSeconds = &durationSeconds.Int64
		}
		if alias.Valid {
			hw.Alias = &alias.String
		}

		histories = append(histories, hw)
	}

	return histories, nil
}

func (db *DB) ListHistoryByHost(hostID int64, limit int) ([]*HistoryWithHost, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT h.id, h.host_id, h.started_at, h.ended_at, h.duration_seconds,
		       hs.hostname, hs.alias
		FROM history h
		JOIN hosts hs ON h.host_id = hs.id
		WHERE h.host_id = ?
		ORDER BY h.started_at DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, hostID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list history by host: %w", err)
	}
	defer rows.Close()

	var histories []*HistoryWithHost
	for rows.Next() {
		hw := &HistoryWithHost{}
		var endedAt sql.NullTime
		var durationSeconds sql.NullInt64
		var alias sql.NullString

		err := rows.Scan(
			&hw.ID, &hw.HostID, &hw.StartedAt, &endedAt, &durationSeconds,
			&hw.Hostname, &alias,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}

		if endedAt.Valid {
			hw.EndedAt = &endedAt.Time
		}
		if durationSeconds.Valid {
			hw.DurationSeconds = &durationSeconds.Int64
		}
		if alias.Valid {
			hw.Alias = &alias.String
		}

		histories = append(histories, hw)
	}

	return histories, nil
}

func (db *DB) GetHostAccessStats(hostID int64) (totalSessions int64, totalDuration int64, lastAccessed *time.Time, err error) {
	err = db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(duration_seconds), 0), MAX(started_at)
		FROM history WHERE host_id = ?
	`, hostID).Scan(&totalSessions, &totalDuration, &lastAccessed)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to get host access stats: %w", err)
	}
	return
}

func (db *DB) DeleteHistory(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM history WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete history: %w", err)
	}
	return nil
}

func (db *DB) DeleteHistoryByHost(hostID int64) error {
	_, err := db.conn.Exec(`DELETE FROM history WHERE host_id = ?`, hostID)
	if err != nil {
		return fmt.Errorf("failed to delete history by host: %w", err)
	}
	return nil
}
