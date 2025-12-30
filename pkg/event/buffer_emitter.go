package event

import (
	"context"
	"sync"
	"time"
)

// BufferingEmitter collects events in memory for later processing (e.g.,
// marshaling into a single HTML report). It is safe for concurrent use.
type BufferingEmitter struct {
	mu     sync.Mutex
	events []Event
}

// NewBufferingEmitter returns a new BufferingEmitter.
func NewBufferingEmitter() *BufferingEmitter {
	return &BufferingEmitter{events: make([]Event, 0)}
}

func (b *BufferingEmitter) Emit(_ context.Context, e Event) error {
	// ensure timestamp
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	b.mu.Lock()
	b.events = append(b.events, e)
	b.mu.Unlock()
	return nil
}

// Events returns a copy of collected events.
func (b *BufferingEmitter) Events() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Event, len(b.events))
	copy(out, b.events)
	return out
}
