package analyzer

import (
	"sort"
	"strings"
	"time"
)

// EventCorrelator correlates events to find related issues
type EventCorrelator struct {
	events    []Event
	timeWindow time.Duration
}

// NewEventCorrelator creates a new event correlator
func NewEventCorrelator(events []Event) *EventCorrelator {
	return &EventCorrelator{
		events:    events,
		timeWindow: 5 * time.Minute,
	}
}

// SetTimeWindow sets the time window for correlation
func (c *EventCorrelator) SetTimeWindow(d time.Duration) {
	c.timeWindow = d
}

// FindRelatedEvents finds events related to a specific resource
func (c *EventCorrelator) FindRelatedEvents(resourceUID string) []Event {
	var related []Event
	for _, e := range c.events {
		if e.ResourceUID == resourceUID {
			related = append(related, e)
		}
	}

	// Sort by time (newest first)
	sort.Slice(related, func(i, j int) bool {
		return related[i].LastSeen.After(related[j].LastSeen)
	})

	return related
}

// FindEventsInTimeWindow finds events within a time window
func (c *EventCorrelator) FindEventsInTimeWindow(start, end time.Time) []Event {
	var matching []Event
	for _, e := range c.events {
		if (e.LastSeen.After(start) || e.LastSeen.Equal(start)) &&
			(e.LastSeen.Before(end) || e.LastSeen.Equal(end)) {
			matching = append(matching, e)
		}
	}
	return matching
}

// FindWarningEvents returns only warning events
func (c *EventCorrelator) FindWarningEvents() []Event {
	var warnings []Event
	for _, e := range c.events {
		if e.Type == "Warning" {
			warnings = append(warnings, e)
		}
	}
	return warnings
}

// FindEventsByReason finds events by reason
func (c *EventCorrelator) FindEventsByReason(reason string) []Event {
	var matching []Event
	for _, e := range c.events {
		if e.Reason == reason {
			matching = append(matching, e)
		}
	}
	return matching
}

// FindEventsByPattern finds events matching a pattern in the message
func (c *EventCorrelator) FindEventsByPattern(pattern string) []Event {
	pattern = strings.ToLower(pattern)
	var matching []Event
	for _, e := range c.events {
		if strings.Contains(strings.ToLower(e.Message), pattern) {
			matching = append(matching, e)
		}
	}
	return matching
}

// CorrelateByTime groups events that occurred close together
func (c *EventCorrelator) CorrelateByTime() [][]Event {
	if len(c.events) == 0 {
		return nil
	}

	// Sort events by time
	sorted := make([]Event, len(c.events))
	copy(sorted, c.events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastSeen.Before(sorted[j].LastSeen)
	})

	var groups [][]Event
	var currentGroup []Event

	for i, e := range sorted {
		if i == 0 {
			currentGroup = append(currentGroup, e)
			continue
		}

		// If event is within time window of previous, add to group
		if e.LastSeen.Sub(sorted[i-1].LastSeen) <= c.timeWindow {
			currentGroup = append(currentGroup, e)
		} else {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = []Event{e}
		}
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// GetEventChain builds a chain of events for a resource
func (c *EventCorrelator) GetEventChain(resourceUID string) []Event {
	related := c.FindRelatedEvents(resourceUID)

	// Sort chronologically
	sort.Slice(related, func(i, j int) bool {
		return related[i].FirstSeen.Before(related[j].FirstSeen)
	})

	return related
}

// GetEventSummary returns a summary of events
func (c *EventCorrelator) GetEventSummary() map[string]int {
	summary := make(map[string]int)
	for _, e := range c.events {
		summary[e.Reason]++
	}
	return summary
}
