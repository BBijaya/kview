package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)

	store := &SQLiteStore{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	_, err := s.db.Exec(Schema)
	return err
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// SaveEvent saves or updates an event
func (s *SQLiteStore) SaveEvent(ctx context.Context, event *Event) error {
	query := `
		INSERT INTO events (id, cluster_id, namespace, resource_uid, resource_kind,
			resource_name, type, reason, message, count, first_seen, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			count = count + 1,
			last_seen = excluded.last_seen,
			message = excluded.message
	`

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.ClusterID, event.Namespace, event.ResourceUID,
		event.ResourceKind, event.ResourceName, event.Type, event.Reason,
		event.Message, event.Count, event.FirstSeen, event.LastSeen,
	)
	return err
}

// GetEvents retrieves events matching the filter
func (s *SQLiteStore) GetEvents(ctx context.Context, filter EventFilter) ([]Event, error) {
	query := "SELECT id, cluster_id, namespace, resource_uid, resource_kind, resource_name, type, reason, message, count, first_seen, last_seen FROM events WHERE 1=1"
	var args []interface{}

	if filter.ClusterID != "" {
		query += " AND cluster_id = ?"
		args = append(args, filter.ClusterID)
	}
	if filter.Namespace != "" {
		query += " AND namespace = ?"
		args = append(args, filter.Namespace)
	}
	if filter.ResourceUID != "" {
		query += " AND resource_uid = ?"
		args = append(args, filter.ResourceUID)
	}
	if filter.ResourceKind != "" {
		query += " AND resource_kind = ?"
		args = append(args, filter.ResourceKind)
	}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Reason != "" {
		query += " AND reason = ?"
		args = append(args, filter.Reason)
	}
	if !filter.Since.IsZero() {
		query += " AND last_seen >= ?"
		args = append(args, filter.Since)
	}
	if !filter.Until.IsZero() {
		query += " AND last_seen <= ?"
		args = append(args, filter.Until)
	}

	query += " ORDER BY last_seen DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(
			&e.ID, &e.ClusterID, &e.Namespace, &e.ResourceUID,
			&e.ResourceKind, &e.ResourceName, &e.Type, &e.Reason,
			&e.Message, &e.Count, &e.FirstSeen, &e.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// SaveSnapshot saves a snapshot
func (s *SQLiteStore) SaveSnapshot(ctx context.Context, snapshot *Snapshot) error {
	query := `
		INSERT INTO snapshots (id, cluster_id, namespace, kind, timestamp, data)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		snapshot.ID, snapshot.ClusterID, snapshot.Namespace,
		snapshot.Kind, snapshot.Timestamp, snapshot.Data,
	)
	return err
}

// GetSnapshot retrieves a snapshot by ID
func (s *SQLiteStore) GetSnapshot(ctx context.Context, id string) (*Snapshot, error) {
	query := "SELECT id, cluster_id, namespace, kind, timestamp, data FROM snapshots WHERE id = ?"

	var snap Snapshot
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&snap.ID, &snap.ClusterID, &snap.Namespace,
		&snap.Kind, &snap.Timestamp, &snap.Data,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// GetSnapshotsInRange retrieves snapshots within a time range
func (s *SQLiteStore) GetSnapshotsInRange(ctx context.Context, start, end time.Time) ([]Snapshot, error) {
	query := "SELECT id, cluster_id, namespace, kind, timestamp, data FROM snapshots WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp"

	rows, err := s.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []Snapshot
	for rows.Next() {
		var snap Snapshot
		err := rows.Scan(
			&snap.ID, &snap.ClusterID, &snap.Namespace,
			&snap.Kind, &snap.Timestamp, &snap.Data,
		)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snap)
	}

	return snapshots, rows.Err()
}

// RecordChange records a resource change
func (s *SQLiteStore) RecordChange(ctx context.Context, change *ChangeRecord) error {
	query := `
		INSERT INTO changes (id, cluster_id, resource_uid, resource_kind, resource_name,
			namespace, change_type, before_data, after_data, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		change.ID, change.ClusterID, change.ResourceUID, change.ResourceKind,
		change.ResourceName, change.Namespace, change.ChangeType,
		change.Before, change.After, change.Timestamp,
	)
	return err
}

// GetChanges retrieves changes matching the filter
func (s *SQLiteStore) GetChanges(ctx context.Context, filter ChangeFilter) ([]ChangeRecord, error) {
	query := "SELECT id, cluster_id, resource_uid, resource_kind, resource_name, namespace, change_type, before_data, after_data, timestamp FROM changes WHERE 1=1"
	var args []interface{}

	if filter.ClusterID != "" {
		query += " AND cluster_id = ?"
		args = append(args, filter.ClusterID)
	}
	if filter.ResourceUID != "" {
		query += " AND resource_uid = ?"
		args = append(args, filter.ResourceUID)
	}
	if filter.ResourceKind != "" {
		query += " AND resource_kind = ?"
		args = append(args, filter.ResourceKind)
	}
	if filter.Namespace != "" {
		query += " AND namespace = ?"
		args = append(args, filter.Namespace)
	}
	if filter.ChangeType != "" {
		query += " AND change_type = ?"
		args = append(args, filter.ChangeType)
	}
	if !filter.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}
	if !filter.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.Until)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []ChangeRecord
	for rows.Next() {
		var c ChangeRecord
		err := rows.Scan(
			&c.ID, &c.ClusterID, &c.ResourceUID, &c.ResourceKind,
			&c.ResourceName, &c.Namespace, &c.ChangeType,
			&c.Before, &c.After, &c.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}

	return changes, rows.Err()
}

// GetSetting retrieves a setting value
func (s *SQLiteStore) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting saves a setting value
func (s *SQLiteStore) SetSetting(ctx context.Context, key, value string) error {
	query := "INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
	_, err := s.db.ExecContext(ctx, query, key, value)
	return err
}

// Cleanup removes old data beyond the retention period
func (s *SQLiteStore) Cleanup(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)

	// Cleanup old events
	_, err := s.db.ExecContext(ctx, "DELETE FROM events WHERE last_seen < ?", cutoff)
	if err != nil {
		return err
	}

	// Cleanup old snapshots
	_, err = s.db.ExecContext(ctx, "DELETE FROM snapshots WHERE timestamp < ?", cutoff)
	if err != nil {
		return err
	}

	// Cleanup old changes
	_, err = s.db.ExecContext(ctx, "DELETE FROM changes WHERE timestamp < ?", cutoff)
	if err != nil {
		return err
	}

	// Vacuum to reclaim space
	_, err = s.db.ExecContext(ctx, "VACUUM")
	return err
}
