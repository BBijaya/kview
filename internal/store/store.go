package store

import (
	"context"
	"time"
)

// Store defines the interface for persistence
type Store interface {
	// Events
	SaveEvent(ctx context.Context, event *Event) error
	GetEvents(ctx context.Context, filter EventFilter) ([]Event, error)

	// Snapshots
	SaveSnapshot(ctx context.Context, snapshot *Snapshot) error
	GetSnapshot(ctx context.Context, id string) (*Snapshot, error)
	GetSnapshotsInRange(ctx context.Context, start, end time.Time) ([]Snapshot, error)

	// Changes
	RecordChange(ctx context.Context, change *ChangeRecord) error
	GetChanges(ctx context.Context, filter ChangeFilter) ([]ChangeRecord, error)

	// Lifecycle
	Close() error
}

// Event represents a Kubernetes event stored in the database
type Event struct {
	ID           string
	ClusterID    string
	Namespace    string
	ResourceUID  string
	ResourceKind string
	ResourceName string
	Type         string // Normal, Warning
	Reason       string
	Message      string
	Count        int
	FirstSeen    time.Time
	LastSeen     time.Time
}

// Snapshot represents a point-in-time snapshot of resources
type Snapshot struct {
	ID        string
	ClusterID string
	Namespace string
	Kind      string
	Timestamp time.Time
	Data      []byte // JSON-encoded resources
}

// ChangeRecord represents a recorded change to a resource
type ChangeRecord struct {
	ID           string
	ClusterID    string
	ResourceUID  string
	ResourceKind string
	ResourceName string
	Namespace    string
	ChangeType   string // Created, Updated, Deleted
	Before       []byte // JSON
	After        []byte // JSON
	Timestamp    time.Time
}

// EventFilter defines filters for querying events
type EventFilter struct {
	ClusterID    string
	Namespace    string
	ResourceUID  string
	ResourceKind string
	Type         string
	Reason       string
	Since        time.Time
	Until        time.Time
	Limit        int
}

// ChangeFilter defines filters for querying changes
type ChangeFilter struct {
	ClusterID    string
	ResourceUID  string
	ResourceKind string
	Namespace    string
	ChangeType   string
	Since        time.Time
	Until        time.Time
	Limit        int
}

// ChangeType constants
const (
	ChangeTypeCreated = "Created"
	ChangeTypeUpdated = "Updated"
	ChangeTypeDeleted = "Deleted"
)
