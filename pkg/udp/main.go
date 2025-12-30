package udp

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
	Emitter    event.Emitter
	Dry        bool
	Timeout    time.Duration
	Data       io.Reader
	RecvBuffer int
}

// WithEmitter sets a custom emitter.
func WithEmitter(e event.Emitter) Option { return func(c *traceConfig) { c.Emitter = e } }

// WithDryRun enables dry-run mode.
func WithDryRun(d bool) Option { return func(c *traceConfig) { c.Dry = d } }

// WithTimeout sets the read/write timeout.
func WithTimeout(d time.Duration) Option { return func(c *traceConfig) { c.Timeout = d } }

// WithData sets a payload to send.
func WithData(r io.Reader) Option { return func(c *traceConfig) { c.Data = r } }

// WithDataString convenience helper.
func WithDataString(s string) Option { return func(c *traceConfig) { c.Data = strings.NewReader(s) } }

// WithRecvBuffer sets a custom buffer size for reads.
func WithRecvBuffer(n int) Option { return func(c *traceConfig) { c.RecvBuffer = n } }

// TraceAddr sends a UDP packet to addr (host:port) and optionally waits for a response.
func TraceAddr(ctx context.Context, addr string, opts ...Option) error {
	cfg := &traceConfig{Timeout: 5 * time.Second, RecvBuffer: 4096}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.Emitter == nil {
		cfg.Emitter = event.NewStdoutEmitter(os.Stdout, true, true)
	}

	traceID := uuid.NewString()
	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "request_start", TraceID: traceID, Payload: map[string]interface{}{"addr": addr}})

	if cfg.Dry {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "dry_run", TraceID: traceID})
		return nil
	}

	// Resolve address and dial UDP
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "error", Stage: "resolve_error", TraceID: traceID, Payload: map[string]interface{}{"error": err.Error()}})
		return err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "error", Stage: "dial_error", TraceID: traceID, Payload: map[string]interface{}{"error": err.Error()}})
		return err
	}
	defer conn.Close()

	connID := uuid.NewString()
	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "connected", TraceID: traceID, ConnID: connID, Payload: map[string]interface{}{"remote": conn.RemoteAddr().String()}})

	// send data
	if cfg.Data != nil {
		buf := make([]byte, 0)
		n, _ := cfg.Data.Read(buf)
		// reading into zero-length will be zero; instead copy by io.Copy to a buffer
		// fallback: if Data is strings.Reader, we can re-read via io.ReadAll
		payload, _ := io.ReadAll(cfg.Data)
		nn, _ := conn.Write(payload)
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "data_send", TraceID: traceID, ConnID: connID, Payload: map[string]interface{}{"bytes_sent": nn}})

		// wait for response with deadline
		_ = conn.SetReadDeadline(time.Now().Add(cfg.Timeout))
		rbuf := make([]byte, cfg.RecvBuffer)
		rn, rerr := conn.Read(rbuf)
		if rn > 0 {
			cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "data_recv", TraceID: traceID, ConnID: connID, Payload: map[string]interface{}{"bytes_recv": rn}})
		}
		if rerr != nil {
			// timeout or other error — emit as lifecycle with error payload
			cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "recv_error", TraceID: traceID, ConnID: connID, Payload: map[string]interface{}{"error": rerr.Error()}})
		}
		_ = n
	}

	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "request_end", TraceID: traceID, ConnID: connID})

	return nil
}
