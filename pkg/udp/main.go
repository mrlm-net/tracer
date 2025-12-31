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
	"github.com/mrlm-net/tracer/pkg/netutil"
	"github.com/mrlm-net/tracer/pkg/tracecommon"
)

type Option func(*traceConfig)

type traceConfig struct {
	Emitter    event.Emitter
	Dry        bool
	Timeout    time.Duration
	Data       io.Reader
	RecvBuffer int
	IPPref     string
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

// WithIPPreference sets IP family preference: "v4", "v6" or ""/"auto".
func WithIPPreference(p string) Option { return func(c *traceConfig) { c.IPPref = p } }

// TraceAddr sends a UDP packet to addr (host:port) and optionally waits for a response.
func TraceAddr(ctx context.Context, addr string, opts ...Option) error {
	cfg := &traceConfig{Timeout: 5 * time.Second, RecvBuffer: 4096}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.Emitter == nil {
		cfg.Emitter = event.NewStdoutEmitter(os.Stdout, true, true)
	}

	traceID := tracecommon.StartRequest(ctx, cfg.Emitter, "udp", addr)
	if cfg.Dry {
		tracecommon.EmitDryRun(ctx, cfg.Emitter, "udp", traceID)
		return nil
	}

	// Parse and dial with IP-family awareness
	host, port, joinAddr, ip, isIP, _, perr := netutil.ParseAddr(addr, "80")
	if perr != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "error", Stage: "resolve_error", TraceID: traceID, Payload: map[string]interface{}{"error": perr.Error()}})
		return perr
	}

	var conn net.Conn
	var chosenIP net.IP
	var resolved []net.IP
	var fam string
	var derr error

	if isIP {
		if netutil.IsIPv4(ip) {
			conn, derr = (&net.Dialer{Timeout: cfg.Timeout}).DialContext(ctx, "udp4", joinAddr)
			fam = "v4"
		} else {
			conn, derr = (&net.Dialer{Timeout: cfg.Timeout}).DialContext(ctx, "udp6", joinAddr)
			fam = "v6"
		}
		chosenIP = ip
	} else {
		conn, chosenIP, resolved, fam, derr = netutil.ResolveAndDial(ctx, "udp", host, port, cfg.IPPref, cfg.Timeout)
	}

	if derr != nil {
		tracecommon.EmitError(ctx, cfg.Emitter, "udp", "dial_error", traceID, derr)
		return derr
	}
	defer conn.Close()

	connID := uuid.NewString()
	tags := tracecommon.BuildTags(chosenIP, resolved, fam)
	tracecommon.EmitLifecycle(ctx, cfg.Emitter, "udp", "connected", traceID, connID, 0, tags, map[string]interface{}{"remote": conn.RemoteAddr().String()})

	// send data
	if cfg.Data != nil {
		payload, _ := io.ReadAll(cfg.Data)
		nn, _ := conn.Write(payload)
		tracecommon.EmitLifecycle(ctx, cfg.Emitter, "udp", "data_send", traceID, connID, 0, nil, map[string]interface{}{"bytes_sent": nn})

		// wait for response with deadline
		_ = conn.SetReadDeadline(time.Now().Add(cfg.Timeout))
		rbuf := make([]byte, cfg.RecvBuffer)
		rn, rerr := conn.Read(rbuf)
		if rn > 0 {
			tracecommon.EmitLifecycle(ctx, cfg.Emitter, "udp", "data_recv", traceID, connID, 0, nil, map[string]interface{}{"bytes_recv": rn})
		}
		if rerr != nil {
			// timeout or other error — emit as lifecycle with error payload
			tracecommon.EmitLifecycle(ctx, cfg.Emitter, "udp", "recv_error", traceID, connID, 0, nil, map[string]interface{}{"error": rerr.Error()})
		}

	}

	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "udp", EventType: "lifecycle", Stage: "request_end", TraceID: traceID, ConnID: connID})

	return nil
}
