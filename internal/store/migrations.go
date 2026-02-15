package store

// Schema contains the SQL statements for creating the database schema
const Schema = `
-- Events table stores Kubernetes events
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    namespace TEXT,
    resource_uid TEXT,
    resource_kind TEXT,
    resource_name TEXT,
    type TEXT,
    reason TEXT,
    message TEXT,
    count INTEGER DEFAULT 1,
    first_seen DATETIME,
    last_seen DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_cluster ON events(cluster_id);
CREATE INDEX IF NOT EXISTS idx_events_resource ON events(resource_uid);
CREATE INDEX IF NOT EXISTS idx_events_namespace ON events(namespace);
CREATE INDEX IF NOT EXISTS idx_events_time ON events(last_seen);

-- Snapshots table stores point-in-time state
CREATE TABLE IF NOT EXISTS snapshots (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    namespace TEXT,
    kind TEXT,
    timestamp DATETIME NOT NULL,
    data BLOB,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_snapshots_cluster ON snapshots(cluster_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_time ON snapshots(timestamp);
CREATE INDEX IF NOT EXISTS idx_snapshots_kind ON snapshots(kind);

-- Changes table stores resource change history
CREATE TABLE IF NOT EXISTS changes (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    resource_uid TEXT NOT NULL,
    resource_kind TEXT,
    resource_name TEXT,
    namespace TEXT,
    change_type TEXT,
    before_data BLOB,
    after_data BLOB,
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_changes_cluster ON changes(cluster_id);
CREATE INDEX IF NOT EXISTS idx_changes_resource ON changes(resource_uid);
CREATE INDEX IF NOT EXISTS idx_changes_namespace ON changes(namespace);
CREATE INDEX IF NOT EXISTS idx_changes_time ON changes(timestamp);
CREATE INDEX IF NOT EXISTS idx_changes_type ON changes(change_type);

-- Settings table stores application settings
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Metadata table for schema versioning
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- Insert schema version
INSERT OR REPLACE INTO metadata (key, value) VALUES ('schema_version', '1');
`

// Migrations contains migration statements for schema updates
var Migrations = map[int]string{
	// Version 1 is the initial schema (defined above)
	1: "", // No migration needed for initial version

	// Future migrations go here
	// 2: `ALTER TABLE events ADD COLUMN source TEXT;`,
}
