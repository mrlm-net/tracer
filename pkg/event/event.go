package event

import (
	"context"
	"time"
)

// Event is a normalized trace event that can represent HTTP/TCP/UDP lifecycle data.
type Event struct {
	Timestamp  time.Time              `json:"timestamp"`
	Protocol   string                 `json:"protocol,omitempty"`   // http|tcp|udp
	EventType  string                 `json:"event_type,omitempty"` // lifecycle|metric|error
	Stage      string                 `json:"stage,omitempty"`      // dns_start, connect_done, response_headers, etc.
	TraceID    string                 `json:"trace_id,omitempty"`
	ConnID     string                 `json:"conn_id,omitempty"`
	DurationNS int64                  `json:"duration_ns,omitempty"`
	Tags       map[string]string      `json:"tags,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// Emitter receives normalized events and forwards them to sinks (stdout, file, remote).
type Emitter interface {
	Emit(ctx context.Context, e Event) error
}
