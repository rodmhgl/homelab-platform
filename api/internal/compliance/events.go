package compliance

import (
	"strings"
	"sync"
	"time"
)

// EventStore manages security events from Falco
// This is the persistence layer for runtime security events
type EventStore interface {
	Add(event SecurityEvent)
	List(filters EventFilters) []SecurityEvent
}

// EventFilters for querying security events
type EventFilters struct {
	Namespace string    // Filter by namespace (checks if resource contains "namespace/")
	Severity  string    // Filter by severity (exact match: Warning, Error, Critical)
	Rule      string    // Filter by rule name (exact match)
	Since     time.Time // Only return events after this timestamp
	Limit     int       // Maximum number of events to return (0 = unlimited)
}

// inMemoryEventStore stores events in a circular buffer
// This is suitable for a homelab demo; production would use PostgreSQL or etcd
type inMemoryEventStore struct {
	mu      sync.RWMutex
	events  []SecurityEvent
	maxSize int
}

// NewInMemoryEventStore creates a new in-memory event store with a circular buffer
// maxSize controls the maximum number of events to retain
func NewInMemoryEventStore(maxSize int) EventStore {
	return &inMemoryEventStore{
		events:  make([]SecurityEvent, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add appends a new security event to the store
// When the buffer is full, the oldest event is dropped (circular buffer)
func (s *inMemoryEventStore) Add(event SecurityEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	// Circular buffer: drop oldest events when exceeding maxSize
	if len(s.events) > s.maxSize {
		s.events = s.events[len(s.events)-s.maxSize:]
	}
}

// List returns events matching the provided filters
// Events are returned in chronological order (oldest first)
func (s *inMemoryEventStore) List(filters EventFilters) []SecurityEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SecurityEvent, 0)

	for _, event := range s.events {
		if matchesFilters(event, filters) {
			result = append(result, event)
		}
	}

	// Apply limit (return most recent N events)
	if filters.Limit > 0 && len(result) > filters.Limit {
		result = result[len(result)-filters.Limit:]
	}

	return result
}

// matchesFilters checks if a security event matches all provided filters
func matchesFilters(event SecurityEvent, filters EventFilters) bool {
	// Filter by namespace (check if resource contains "namespace/")
	if filters.Namespace != "" && !strings.Contains(event.Resource, filters.Namespace+"/") {
		return false
	}

	// Filter by severity (exact match, case-sensitive)
	if filters.Severity != "" && event.Severity != filters.Severity {
		return false
	}

	// Filter by rule name (exact match)
	if filters.Rule != "" && event.Rule != filters.Rule {
		return false
	}

	// Filter by timestamp (only return events after Since)
	if !filters.Since.IsZero() {
		eventTime, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			// Skip events with malformed timestamps
			return false
		}
		if eventTime.Before(filters.Since) {
			return false
		}
	}

	return true
}
