package event

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// StdoutEmitter writes events to an io.Writer. It supports NDJSON (machine) and
// a human-friendly pretty mode. Both can be enabled: pretty summary will be
// printed first, then the JSON line to allow programmatic parsing.
type StdoutEmitter struct {
	w      io.Writer
	ndjson bool
	pretty bool
	mu     sync.Mutex
	enc    *json.Encoder
}

// NewStdoutEmitter creates a StdoutEmitter. If ndjson is true it will emit
// newline-delimited JSON for each event. If pretty is true it will also print
// a short human summary before the JSON line.
func NewStdoutEmitter(w io.Writer, ndjson bool, pretty bool) *StdoutEmitter {
	return &StdoutEmitter{w: w, ndjson: ndjson, pretty: pretty, enc: json.NewEncoder(w)}
}

func (s *StdoutEmitter) Emit(_ context.Context, e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure timestamp
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	if s.pretty {
		summary := fmt.Sprintf("[%s] %s %s %s",
			e.Timestamp.Format(time.RFC3339Nano), e.Protocol, e.Stage, e.EventType)
		if e.TraceID != "" {
			summary = fmt.Sprintf("%s trace=%s", summary, e.TraceID)
		}
		if _, err := fmt.Fprintln(s.w, summary); err != nil {
			return err
		}
	}

	if s.ndjson {
		// Encode as JSON line
		if err := s.enc.Encode(e); err != nil {
			return err
		}
	}

	// If neither mode is enabled, fall back to a simple print
	if !s.ndjson && !s.pretty {
		_, err := fmt.Fprintf(s.w, "%+v\n", e)
		return err
	}

	return nil
}
