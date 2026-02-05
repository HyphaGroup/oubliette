package session

import (
	"sync"
	"testing"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

func TestEventBuffer_Append(t *testing.T) {
	buf := NewEventBuffer("test-session", 10)

	// First event should have index 0
	idx := buf.Append(&agent.StreamEvent{Type: "message", Text: "data1"})
	if idx != 0 {
		t.Errorf("First event index = %v, want 0", idx)
	}

	// Second event should have index 1
	idx = buf.Append(&agent.StreamEvent{Type: "message", Text: "data2"})
	if idx != 1 {
		t.Errorf("Second event index = %v, want 1", idx)
	}

	if buf.Len() != 2 {
		t.Errorf("Len() = %v, want 2", buf.Len())
	}
}

func TestEventBuffer_After(t *testing.T) {
	buf := NewEventBuffer("test-session", 10)

	buf.Append(&agent.StreamEvent{Type: "message", Text: "data0"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data1"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data2"})

	tests := []struct {
		name      string
		index     int
		wantCount int
		wantErr   bool
	}{
		{"all events (since -1)", -1, 3, false},
		{"after first event", 0, 2, false},
		{"after second event", 1, 1, false},
		{"after last event", 2, 0, false},
		{"future index", 100, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := buf.After(tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("After() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(events) != tt.wantCount {
				t.Errorf("After() count = %v, want %v", len(events), tt.wantCount)
			}
		})
	}
}

func TestEventBuffer_RingBufferBehavior(t *testing.T) {
	buf := NewEventBuffer("test-session", 3)

	// Fill buffer
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data0"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data1"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data2"})

	if buf.Len() != 3 {
		t.Errorf("Len() = %v, want 3", buf.Len())
	}

	// Add one more - should drop oldest
	idx := buf.Append(&agent.StreamEvent{Type: "message", Text: "data3"})
	if idx != 3 {
		t.Errorf("Fourth event index = %v, want 3", idx)
	}

	if buf.Len() != 3 {
		t.Errorf("Len() = %v, want 3 (max size)", buf.Len())
	}

	if buf.StartIndex() != 1 {
		t.Errorf("StartIndex() = %v, want 1 (oldest dropped)", buf.StartIndex())
	}

	if buf.DroppedEvents() != 1 {
		t.Errorf("DroppedEvents() = %v, want 1", buf.DroppedEvents())
	}

	// After index -1 should return all 3 remaining events
	events, err := buf.After(-1)
	if err != nil {
		t.Fatalf("After(-1) error = %v", err)
	}
	if len(events) != 3 {
		t.Errorf("After(-1) count = %v, want 3", len(events))
	}

	// Verify content - should be data1, data2, data3
	expectedData := []string{"data1", "data2", "data3"}
	for i, e := range events {
		if e.Event.Text != expectedData[i] {
			t.Errorf("events[%d].Text = %v, want %v", i, e.Event.Text, expectedData[i])
		}
	}
}

func TestEventBuffer_PurgedEventsError(t *testing.T) {
	buf := NewEventBuffer("test-session", 2)

	// Fill buffer with 4 events (drops first 2)
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data0"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data1"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data2"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data3"})

	// StartIndex should be 2 now
	if buf.StartIndex() != 2 {
		t.Errorf("StartIndex() = %v, want 2", buf.StartIndex())
	}

	// Requesting index 0 (purged) should error
	_, err := buf.After(0)
	if err == nil {
		t.Error("After(0) should return error for purged events")
	}
}

func TestEventBuffer_LastIndex(t *testing.T) {
	buf := NewEventBuffer("test-session", 10)

	// Empty buffer
	if buf.LastIndex() != -1 {
		t.Errorf("LastIndex() on empty = %v, want -1", buf.LastIndex())
	}

	buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})
	if buf.LastIndex() != 0 {
		t.Errorf("LastIndex() = %v, want 0", buf.LastIndex())
	}

	buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})
	if buf.LastIndex() != 1 {
		t.Errorf("LastIndex() = %v, want 1", buf.LastIndex())
	}
}

func TestEventBuffer_All(t *testing.T) {
	buf := NewEventBuffer("test-session", 10)

	buf.Append(&agent.StreamEvent{Type: "message", Text: "data0"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data1"})

	all := buf.All()
	if len(all) != 2 {
		t.Errorf("All() count = %v, want 2", len(all))
	}
}

func TestEventBuffer_Clear(t *testing.T) {
	buf := NewEventBuffer("test-session", 10)

	buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})

	buf.Clear()

	if buf.Len() != 0 {
		t.Errorf("Len() after Clear() = %v, want 0", buf.Len())
	}
	if buf.StartIndex() != 0 {
		t.Errorf("StartIndex() after Clear() = %v, want 0", buf.StartIndex())
	}
}

func TestEventBuffer_Stats(t *testing.T) {
	buf := NewEventBuffer("test-session", 5)

	buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})
	buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})

	stats := buf.Stats()

	if stats.SessionID != "test-session" {
		t.Errorf("Stats.SessionID = %v, want test-session", stats.SessionID)
	}
	if stats.CurrentSize != 2 {
		t.Errorf("Stats.CurrentSize = %v, want 2", stats.CurrentSize)
	}
	if stats.MaxSize != 5 {
		t.Errorf("Stats.MaxSize = %v, want 5", stats.MaxSize)
	}
	if stats.LastIndex != 1 {
		t.Errorf("Stats.LastIndex = %v, want 1", stats.LastIndex)
	}
}

func TestEventBuffer_DefaultSize(t *testing.T) {
	buf := NewEventBuffer("test-session", 0)

	stats := buf.Stats()
	if stats.MaxSize != DefaultEventBufferSize {
		t.Errorf("Default MaxSize = %v, want %v", stats.MaxSize, DefaultEventBufferSize)
	}
}

func TestEventBuffer_ConcurrentAccess(t *testing.T) {
	buf := NewEventBuffer("test-session", 100)
	var wg sync.WaitGroup

	// Concurrent appends
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			buf.Append(&agent.StreamEvent{Type: "message", Text: "data"})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf.All()
			_, _ = buf.After(-1)
			buf.LastIndex()
			buf.Len()
			buf.Stats()
		}()
	}

	wg.Wait()

	// Should have exactly 50 events
	if buf.Len() != 50 {
		t.Errorf("Len() = %v, want 50", buf.Len())
	}
}

func TestEventBuffer_SessionID(t *testing.T) {
	buf := NewEventBuffer("my-session", 10)
	if buf.SessionID() != "my-session" {
		t.Errorf("SessionID() = %v, want my-session", buf.SessionID())
	}
}
