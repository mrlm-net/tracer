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
	"github.com/mrlm-net/tracer/pkg/netutil"
)

type Option func(*traceConfig)

type traceConfig struct {
	Emitter event.Emitter
	Dry     bool
	Timeout time.Duration
	Data    io.Reader
	IPPref  string
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

// WithIPPreference sets IP family preference: "v4", "v6" or ""/"auto".
func WithIPPreference(p string) Option { return func(c *traceConfig) { c.IPPref = p } }

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

	// Parse and dial with IP-family awareness
	host, port, joinAddr, ip, isIP, _, perr := netutil.ParseAddr(addr, "80")
	if perr != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "error", Stage: "resolve_error", TraceID: traceID, Payload: map[string]interface{}{"error": perr.Error()}})
		return perr
	}

	var conn net.Conn
	var chosenIP net.IP
	var resolved []net.IP
	var fam string
	var derr error

	if isIP {
		// direct dial using explicit family when IP literal provided
		if netutil.IsIPv4(ip) {
			conn, derr = (&net.Dialer{Timeout: cfg.Timeout}).DialContext(ctx, "tcp4", joinAddr)
			fam = "v4"
		} else {
			conn, derr = (&net.Dialer{Timeout: cfg.Timeout}).DialContext(ctx, "tcp6", joinAddr)
			fam = "v6"
		}
		chosenIP = ip
	} else {
		conn, chosenIP, resolved, fam, derr = netutil.ResolveAndDial(ctx, "tcp", host, port, cfg.IPPref, cfg.Timeout)
	}

	if derr != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "error", Stage: "connect_error", TraceID: traceID, Payload: map[string]interface{}{"error": derr.Error()}})
		return derr
	}
	defer conn.Close()

	connID := uuid.NewString()
	// add ip family metadata if available
	tags := map[string]string{}
	if fam != "" {
		tags["ip_family"] = fam
	}
	if chosenIP != nil {
		tags["remote_ip"] = chosenIP.String()
	}
	if len(resolved) > 0 {
		// serialize as comma list
		var sb strings.Builder
		for i, rip := range resolved {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(rip.String())
		}
		tags["resolved_ips"] = sb.String()
	}

	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "tcp", EventType: "lifecycle", Stage: "connect_done", TraceID: traceID, ConnID: connID, DurationNS: int64(time.Since(start)), Tags: tags, Payload: map[string]interface{}{"remote": conn.RemoteAddr().String(), "local": conn.LocalAddr().String()}})

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
