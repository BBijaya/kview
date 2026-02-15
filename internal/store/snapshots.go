package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bijaya/kview/internal/k8s"
)

// SnapshotBuilder helps create snapshots
type SnapshotBuilder struct {
	clusterID string
}

// NewSnapshotBuilder creates a new snapshot builder
func NewSnapshotBuilder(clusterID string) *SnapshotBuilder {
	return &SnapshotBuilder{clusterID: clusterID}
}

// BuildFromResources creates a snapshot from resources
func (b *SnapshotBuilder) BuildFromResources(namespace, kind string, resources []k8s.Resource) (*Snapshot, error) {
	data, err := json.Marshal(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resources: %w", err)
	}

	now := time.Now()
	id := b.generateID(namespace, kind, now)

	return &Snapshot{
		ID:        id,
		ClusterID: b.clusterID,
		Namespace: namespace,
		Kind:      kind,
		Timestamp: now,
		Data:      data,
	}, nil
}

// BuildFromPods creates a snapshot from pod info
func (b *SnapshotBuilder) BuildFromPods(namespace string, pods []k8s.PodInfo) (*Snapshot, error) {
	data, err := json.Marshal(pods)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pods: %w", err)
	}

	now := time.Now()
	id := b.generateID(namespace, "Pod", now)

	return &Snapshot{
		ID:        id,
		ClusterID: b.clusterID,
		Namespace: namespace,
		Kind:      "Pod",
		Timestamp: now,
		Data:      data,
	}, nil
}

func (b *SnapshotBuilder) generateID(namespace, kind string, timestamp time.Time) string {
	data := fmt.Sprintf("%s:%s:%s:%d", b.clusterID, namespace, kind, timestamp.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// DecodeSnapshot decodes snapshot data into resources
func DecodeSnapshot(snap *Snapshot) ([]k8s.Resource, error) {
	if snap == nil || len(snap.Data) == 0 {
		return nil, nil
	}

	var resources []k8s.Resource
	if err := json.Unmarshal(snap.Data, &resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}
	return resources, nil
}

// DecodePodSnapshot decodes snapshot data into pod info
func DecodePodSnapshot(snap *Snapshot) ([]k8s.PodInfo, error) {
	if snap == nil || len(snap.Data) == 0 {
		return nil, nil
	}

	var pods []k8s.PodInfo
	if err := json.Unmarshal(snap.Data, &pods); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pods snapshot: %w", err)
	}
	return pods, nil
}

// SnapshotDiff represents the difference between two snapshots
type SnapshotDiff struct {
	Added    []string // Names of added resources
	Removed  []string // Names of removed resources
	Modified []string // Names of modified resources
}

// DiffSnapshots compares two snapshots and returns the differences
func DiffSnapshots(before, after *Snapshot) (*SnapshotDiff, error) {
	diff := &SnapshotDiff{}

	beforeResources, err := DecodeSnapshot(before)
	if err != nil {
		return nil, err
	}

	afterResources, err := DecodeSnapshot(after)
	if err != nil {
		return nil, err
	}

	// Build maps for comparison
	beforeMap := make(map[string]k8s.Resource)
	for _, r := range beforeResources {
		beforeMap[r.UID] = r
	}

	afterMap := make(map[string]k8s.Resource)
	for _, r := range afterResources {
		afterMap[r.UID] = r
	}

	// Find added and modified
	for uid, afterRes := range afterMap {
		if beforeRes, exists := beforeMap[uid]; exists {
			// Check if modified (simple comparison by fetched time)
			if afterRes.FetchedAt != beforeRes.FetchedAt {
				diff.Modified = append(diff.Modified, afterRes.Name)
			}
		} else {
			diff.Added = append(diff.Added, afterRes.Name)
		}
	}

	// Find removed
	for uid, beforeRes := range beforeMap {
		if _, exists := afterMap[uid]; !exists {
			diff.Removed = append(diff.Removed, beforeRes.Name)
		}
	}

	return diff, nil
}
