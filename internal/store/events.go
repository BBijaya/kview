package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// EventBuilder helps construct events from Kubernetes events
type EventBuilder struct {
	clusterID string
}

// NewEventBuilder creates a new event builder
func NewEventBuilder(clusterID string) *EventBuilder {
	return &EventBuilder{clusterID: clusterID}
}

// Build creates an Event from raw data
func (b *EventBuilder) Build(
	namespace, resourceUID, resourceKind, resourceName,
	eventType, reason, message string,
	count int, firstSeen, lastSeen time.Time,
) *Event {
	// Generate ID from key components
	id := b.generateID(namespace, resourceUID, reason, firstSeen)

	return &Event{
		ID:           id,
		ClusterID:    b.clusterID,
		Namespace:    namespace,
		ResourceUID:  resourceUID,
		ResourceKind: resourceKind,
		ResourceName: resourceName,
		Type:         eventType,
		Reason:       reason,
		Message:      message,
		Count:        count,
		FirstSeen:    firstSeen,
		LastSeen:     lastSeen,
	}
}

func (b *EventBuilder) generateID(namespace, resourceUID, reason string, firstSeen time.Time) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", b.clusterID, namespace, resourceUID, reason, firstSeen.Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// EventStats provides statistics about stored events
type EventStats struct {
	Total     int
	Warnings  int
	Normal    int
	ByReason  map[string]int
	ByKind    map[string]int
	LastEvent time.Time
}

// CalculateStats calculates statistics from a list of events
func CalculateStats(events []Event) EventStats {
	stats := EventStats{
		Total:    len(events),
		ByReason: make(map[string]int),
		ByKind:   make(map[string]int),
	}

	for _, e := range events {
		if e.Type == "Warning" {
			stats.Warnings++
		} else {
			stats.Normal++
		}

		stats.ByReason[e.Reason]++
		stats.ByKind[e.ResourceKind]++

		if e.LastSeen.After(stats.LastEvent) {
			stats.LastEvent = e.LastSeen
		}
	}

	return stats
}

// GroupByResource groups events by resource
func GroupByResource(events []Event) map[string][]Event {
	grouped := make(map[string][]Event)
	for _, e := range events {
		key := fmt.Sprintf("%s/%s/%s", e.ResourceKind, e.Namespace, e.ResourceName)
		grouped[key] = append(grouped[key], e)
	}
	return grouped
}

// GroupByReason groups events by reason
func GroupByReason(events []Event) map[string][]Event {
	grouped := make(map[string][]Event)
	for _, e := range events {
		grouped[e.Reason] = append(grouped[e.Reason], e)
	}
	return grouped
}

// FilterWarnings returns only warning events
func FilterWarnings(events []Event) []Event {
	var warnings []Event
	for _, e := range events {
		if e.Type == "Warning" {
			warnings = append(warnings, e)
		}
	}
	return warnings
}

// FilterByTimeRange filters events within a time range
func FilterByTimeRange(events []Event, start, end time.Time) []Event {
	var filtered []Event
	for _, e := range events {
		if (e.LastSeen.After(start) || e.LastSeen.Equal(start)) &&
			(e.LastSeen.Before(end) || e.LastSeen.Equal(end)) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
