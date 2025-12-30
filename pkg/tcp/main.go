package tcp

import (
	"context"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mrlm-net/tracer/pkg/event"
)

type Option func(*traceConfig)

type traceConfig struct {
	Emitter event.Emitter
	Dry     bool
	Timeout time.Duration
	Data    io.Reader
}

// WithEmitter sets a custom emitter.
func WithEmitter(e event.Emitter) Option { return func(c *traceConfig) { c.Emitter = e } }

// WithDryRun enables dry-run mode.
func WithDryRun(d bool) Option { return func(c *traceConfig) { c.Dry = d } }

// WithTimeout sets the dial/read timeout.
func WithTimeout(d time.Duration) Option { return func(c *traceConfig) { c.Timeout = d } }

// WithData sets a payload to send on the connection.
func WithData(r io.Reader) Option { return func(c *traceConfig) { c.Data = r } }

// WithDataString convenience helper.
func WithDataString(s string) Option { return func(c *traceConfig) { c.Data = strings.NewReader(s) } }

// TraceAddr opens a TCP connection to addr (host:port) and emits events.
func TraceAddr(ctx context.Context, addr string, opts ...Option) error {
	cfg := &traceConfig{Timeout: 30 * time.Second}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.Emitter == nil {
		cfg.Emitter = event.NewStdoutEmitter(os.Stdout, true, true)
	}

	traceID := uuid.NewString()
	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "request_start", TraceID: traceID, Payload: map[string]interface{}{"addr": addr}})

	if cfg.Dry {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "dry_run", TraceID: traceID})
		return nil
	}

	start := time.Now()
	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "connect_start", TraceID: traceID, Payload: map[string]interface{}{"addr": addr}})

	conn, err := net.DialTimeout("tcp", addr, cfg.Timeout)
	if err != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "error", Stage: "connect_error", TraceID: traceID, Payload: map[string]interface{}{"error": err.Error()}})
		return err
	}
	defer conn.Close()

	connID := uuid.NewString()
	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "connect_done", TraceID: traceID, ConnID: connID, DurationNS: int64(time.Since(start)), Payload: map[string]interface{}{"remote": conn.RemoteAddr().String(), "local": conn.LocalAddr().String()}})

	// send data if provided
	if cfg.Data != nil {
		n, _ := io.Copy(conn, cfg.Data)
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "data_send", TraceID: traceID, ConnID: connID, Payload: map[string]interface{}{"bytes_sent": n}})

		// attempt to read a small response
		buf := make([]byte, 1024)
		_ = conn.SetReadDeadline(time.Now().Add(cfg.Timeout))
		nr, _ := conn.Read(buf)
		if nr > 0 {
			cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "data_recv", TraceID: traceID, ConnID: connID, Payload: map[string]interface{}{"bytes_recv": nr}})
		}
	}

	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "request_end", TraceID: traceID, ConnID: connID})

	return nil
}
