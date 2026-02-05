package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

/*
EVENT BUFFER - RING BUFFER FOR STREAMING EVENTS

The EventBuffer provides bounded storage for session events with support for
client disconnect/reconnect via index-based resumption.

DATA STRUCTURE:

    Logical view (indices are monotonically increasing):
    ┌─────────────────────────────────────────────────────────────┐
    │ ... [purged] ... │ startIndex │ event │ event │ ... │ lastIndex │
    └─────────────────────────────────────────────────────────────┘
                        ↑                                    ↑
                        └── oldest buffered event            └── newest event

    Physical storage (slice that grows up to maxSize, then wraps):
    events[0..len-1] maps to logical indices [startIndex..startIndex+len-1]

INDEX MATH:

    - Logical index = startIndex + physical offset
    - Physical offset = logical index - startIndex
    - When buffer full: shift left (drop oldest), increment startIndex

RESUMPTION PROTOCOL:

    Client polls with since_index:
    1. First poll: since_index = -1 (get all buffered events)
    2. Response includes last_index (highest event index returned)
    3. Next poll: since_index = last_index (get events after that)
    4. If client falls too far behind: error "events purged"

WHY RING BUFFER?

    - Bounded memory: Never exceeds maxSize * sizeof(event)
    - Disconnect tolerance: Client can reconnect and resume
    - Simple: No external dependencies (Redis, etc.)
    - Trade-off: Slow clients lose old events (acceptable for UI streaming)

THREAD SAFETY:

    All methods acquire mu (RWMutex):
    - Append: exclusive lock (modifies events slice)
    - After, Len, All: shared lock (read only)

METRICS:

    droppedEvents counter tracks buffer overflow. If this is non-zero,
    clients are not keeping up with event production rate. Options:
    - Increase buffer size (more memory)
    - Speed up client polling
    - Accept data loss for slow clients
*/

// EventBuffer configuration constants
const (
	DefaultEventBufferSize    = 1000
	DefaultSessionIdleTimeout = 30 * time.Minute
	DefaultMaxActiveSessions  = 10
)

// BufferedEvent wraps a stream event with metadata for resumption
type BufferedEvent struct {
	Index     int                `json:"index"`
	Timestamp time.Time          `json:"timestamp"`
	Event     *agent.StreamEvent `json:"event"`
}

// EventBuffer provides a ring buffer for streaming events with resumption support
type EventBuffer struct {
	sessionID     string
	events        []*BufferedEvent
	maxSize       int
	startIndex    int   // Logical index of the first event in the buffer
	droppedEvents int64 // Count of events dropped due to buffer overflow
	mu            sync.RWMutex
}

// BufferStats contains statistics about the event buffer
type BufferStats struct {
	SessionID     string `json:"session_id"`
	CurrentSize   int    `json:"current_size"`
	MaxSize       int    `json:"max_size"`
	StartIndex    int    `json:"start_index"`
	LastIndex     int    `json:"last_index"`
	DroppedEvents int64  `json:"dropped_events"`
}

// NewEventBuffer creates a new event buffer for the given session
func NewEventBuffer(sessionID string, maxSize int) *EventBuffer {
	if maxSize <= 0 {
		maxSize = DefaultEventBufferSize
	}
	return &EventBuffer{
		sessionID:  sessionID,
		events:     make([]*BufferedEvent, 0, maxSize),
		maxSize:    maxSize,
		startIndex: 0,
	}
}

// Append adds an event to the buffer and returns its index
func (b *EventBuffer) Append(event *agent.StreamEvent) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	index := b.startIndex + len(b.events)
	be := &BufferedEvent{
		Index:     index,
		Timestamp: time.Now(),
		Event:     event,
	}

	if len(b.events) >= b.maxSize {
		// Ring buffer - drop oldest event
		b.events = b.events[1:]
		b.startIndex++
		b.droppedEvents++
	}
	b.events = append(b.events, be)
	return index
}

// After returns events after the given index (exclusive)
// Returns error if the requested index has been purged
// Special case: index=-1 returns all available events
func (b *EventBuffer) After(index int) ([]*BufferedEvent, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Special case: -1 means "give me all available events"
	// This is used for first poll when client has no index yet
	if index == -1 {
		result := make([]*BufferedEvent, len(b.events))
		copy(result, b.events)
		return result, nil
	}

	// Check if the requested index is before our buffer window
	if index < b.startIndex-1 {
		return nil, fmt.Errorf("events before index %d have been purged (oldest available: %d)", index, b.startIndex)
	}

	// Calculate the slice offset
	start := index - b.startIndex + 1
	if start < 0 {
		start = 0
	}
	if start >= len(b.events) {
		// No new events after this index
		return []*BufferedEvent{}, nil
	}

	// Copy the slice to avoid holding the lock
	result := make([]*BufferedEvent, len(b.events)-start)
	copy(result, b.events[start:])
	return result, nil
}

// Since returns all events after the given timestamp
func (b *EventBuffer) Since(timestamp time.Time) []*BufferedEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []*BufferedEvent
	for _, e := range b.events {
		if e.Timestamp.After(timestamp) {
			result = append(result, e)
		}
	}
	return result
}

// LastIndex returns the index of the most recent event, or -1 if empty
func (b *EventBuffer) LastIndex() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.events) == 0 {
		return -1
	}
	return b.startIndex + len(b.events) - 1
}

// Len returns the number of events currently buffered
func (b *EventBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.events)
}

// SessionID returns the session ID this buffer belongs to
func (b *EventBuffer) SessionID() string {
	return b.sessionID
}

// All returns all buffered events (for debugging/inspection)
func (b *EventBuffer) All() []*BufferedEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*BufferedEvent, len(b.events))
	copy(result, b.events)
	return result
}

// Clear removes all events from the buffer
func (b *EventBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events = make([]*BufferedEvent, 0, b.maxSize)
	b.startIndex = 0
}

// StartIndex returns the logical index of the first buffered event
func (b *EventBuffer) StartIndex() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.startIndex
}

// DroppedEvents returns the count of events dropped due to buffer overflow
func (b *EventBuffer) DroppedEvents() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.droppedEvents
}

// Stats returns current buffer statistics
func (b *EventBuffer) Stats() BufferStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	lastIndex := -1
	if len(b.events) > 0 {
		lastIndex = b.startIndex + len(b.events) - 1
	}

	return BufferStats{
		SessionID:     b.sessionID,
		CurrentSize:   len(b.events),
		MaxSize:       b.maxSize,
		StartIndex:    b.startIndex,
		LastIndex:     lastIndex,
		DroppedEvents: b.droppedEvents,
	}
}
