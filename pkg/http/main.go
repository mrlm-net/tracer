package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mrlm-net/tracer/pkg/event"
)

type Option func(*traceConfig)

type traceConfig struct {
	Emitter event.Emitter
	Dry     bool
	Timeout time.Duration
	Redact  bool
	// InjectTraceHeader controls whether the tracer will add `X-Trace-Id`
	// to outgoing requests. Default: false.
	InjectTraceHeader bool
	// Method if non-empty overrides the HTTP method used (default GET)
	Method string
	// Body supplies a request body for non-GET methods; may be nil.
	Body io.Reader
	// Headers are additional headers to set on the outgoing request.
	Headers http.Header
}

// WithEmitter sets a custom event.Emitter for TraceURL.
func WithEmitter(e event.Emitter) Option { return func(c *traceConfig) { c.Emitter = e } }

// WithDryRun enables dry-run mode.
func WithDryRun(d bool) Option { return func(c *traceConfig) { c.Dry = d } }

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option { return func(c *traceConfig) { c.Timeout = d } }

// WithInjectTraceHeader controls whether to add X-Trace-Id to requests.
func WithInjectTraceHeader(v bool) Option { return func(c *traceConfig) { c.InjectTraceHeader = v } }

// WithMethod sets the HTTP request method (e.g. POST, PUT, PATCH).
func WithMethod(m string) Option { return func(c *traceConfig) { c.Method = m } }

// WithBody sets the request body for TraceURL. Caller is responsible for
// providing an io.Reader that can be read once by the tracer.
func WithBody(r io.Reader) Option { return func(c *traceConfig) { c.Body = r } }

// WithBodyString is a convenience to set a string body.
func WithBodyString(s string) Option {
	return func(c *traceConfig) { c.Body = bytes.NewBufferString(s) }
}

// WithHeaders sets extra headers on the outgoing request.
func WithHeaders(h http.Header) Option { return func(c *traceConfig) { c.Headers = h } }

// TraceURL performs an HTTP request to targetURL and emits normalized events via the configured Emitter.
// By default it performs a GET; use WithMethod/WithBody/WithHeaders to customize.
func TraceURL(ctx context.Context, targetURL string, opts ...Option) error {
	cfg := &traceConfig{Timeout: 30 * time.Second, Redact: true}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.Emitter == nil {
		cfg.Emitter = event.NewStdoutEmitter(os.Stdout, true, true)
	}

	// simple trace id (UUID)
	traceID := uuid.NewString()

	// emit request_start
	cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: "request_start", TraceID: traceID, Payload: map[string]interface{}{"url": targetURL}})

	if cfg.Dry {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: "dry_run", TraceID: traceID})
		return nil
	}

	method := http.MethodGet
	if cfg.Method != "" {
		method = cfg.Method
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, cfg.Body)
	if err != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "error", Stage: "request_new", TraceID: traceID, Payload: map[string]interface{}{"error": err.Error()}})
		return err
	}

	// attach any provided headers
	if cfg.Headers != nil {
		for k, v := range cfg.Headers {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
	}

	start := time.Now()

	var mu sync.Mutex
	stageStarts := make(map[string]time.Time)

	recordStageStart := func(key string) bool {
		mu.Lock()
		defer mu.Unlock()
		if _, exists := stageStarts[key]; !exists {
			stageStarts[key] = time.Now()
			return true
		}
		return false
	}

	emitStageDone := func(key, stage string, payload map[string]interface{}) {
		mu.Lock()
		s, ok := stageStarts[key]
		if ok {
			delete(stageStarts, key)
		}
		mu.Unlock()
		var d time.Duration
		if ok {
			d = time.Since(s)
		} else {
			d = time.Since(start)
		}
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: stage, TraceID: traceID, DurationNS: int64(d), Payload: payload})
	}

	emit := func(stage string, payload map[string]interface{}) {
		// general emit uses overall request elapsed time
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: stage, TraceID: traceID, DurationNS: int64(time.Since(start)), Payload: payload})
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			// record DNS start (only emit once per roundtrip)
			if recordStageStart("dns") {
				emit("dns_start", map[string]interface{}{"host": info.Host})
			}
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			emitStageDone("dns", "dns_done", map[string]interface{}{"addrs": info.Addrs})
		},
		ConnectStart: func(network, addr string) {
			if recordStageStart("connect:" + addr) {
				emit("connect_start", map[string]interface{}{"network": network, "addr": addr})
			}
		},
		ConnectDone: func(network, addr string, err error) {
			emitStageDone("connect:"+addr, "connect_done", map[string]interface{}{"network": network, "addr": addr, "error": errorString(err)})
		},
		GotConn: func(info httptrace.GotConnInfo) {
			emit("got_conn", map[string]interface{}{"reused": info.Reused, "was_idle": info.WasIdle})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			emit("wrote_request", map[string]interface{}{"error": errorString(info.Err)})
		},
		TLSHandshakeStart: func() {
			if recordStageStart("tls") {
				emit("tls_handshake_start", nil)
			}
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			emitStageDone("tls", "tls_handshake_done", map[string]interface{}{"negotiated_proto": cs.NegotiatedProtocol, "cipher_suite": cs.CipherSuite, "err": errorString(err)})
		},
		GotFirstResponseByte: func() { emit("got_first_response_byte", nil) },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Wrap the transport to capture per-hop request/response headers
	transport := &tracingTransport{base: http.DefaultTransport, emitter: cfg.Emitter, traceID: traceID, redact: cfg.Redact, injectTraceHeader: cfg.InjectTraceHeader}

	client := &http.Client{Timeout: cfg.Timeout, Transport: transport}

	// CheckRedirect allows us to emit redirect events and preserve trace context
	client.CheckRedirect = func(newReq *http.Request, via []*http.Request) error {
		from := ""
		if len(via) > 0 {
			from = via[len(via)-1].URL.String()
		}
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: "redirect", TraceID: traceID, Payload: map[string]interface{}{"from": from, "to": newReq.URL.String()}})

		// propagate previous ClientTrace to new request so callbacks continue
		// copy the ClientTrace value before attaching it to avoid a self-referential
		// composition which can lead to infinite recursion when the trace is
		// already present in the context chain.
		if len(via) > 0 {
			if prev := httptrace.ContextClientTrace(via[len(via)-1].Context()); prev != nil {
				p := *prev
				newReq = newReq.WithContext(httptrace.WithClientTrace(newReq.Context(), &p))
			}
		}

		// ensure redirect request carries the trace id header if configured
		if cfg.InjectTraceHeader {
			if newReq.Header.Get("X-Trace-Id") == "" {
				newReq.Header.Set("X-Trace-Id", traceID)
			}
		}
		return nil // follow redirects
	}

	resp, err := client.Do(req)
	if err != nil {
		cfg.Emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "error", Stage: "request_do", TraceID: traceID, Payload: map[string]interface{}{"error": err.Error()}})
		return err
	}
	defer resp.Body.Close()

	// read small amount of body to ensure response flow
	n, _ := ioCopyNDiscard(resp.Body, 1024)
	emit("response_end", map[string]interface{}{"status": resp.Status, "bytes_read": n})

	return nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ioCopyNDiscard copies up to n bytes from r to nowhere and returns number of bytes read.
func ioCopyNDiscard(r io.Reader, n int64) (int64, error) {
	return io.CopyN(io.Discard, r, n)
}

// tracingTransport wraps a RoundTripper and emits per-hop request/response details.
type tracingTransport struct {
	base              http.RoundTripper
	emitter           event.Emitter
	traceID           string
	redact            bool
	injectTraceHeader bool
}

func (t *tracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	// ensure the request carries the trace id header; clone request when
	// we need to mutate headers so we don't accidentally modify a shared
	// request object used elsewhere.
	r := req
	if t.injectTraceHeader && req.Header.Get("X-Trace-Id") == "" {
		r = req.Clone(ctx)
		r.Header.Set("X-Trace-Id", t.traceID)
	}

	// emit request_send with headers (sanitized)
	reqHdrs := copyHeaders(r.Header)
	if t.redact {
		sanitizeHeaders(reqHdrs, true)
	}
	t.emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: "request_send", TraceID: t.traceID, Payload: map[string]interface{}{"method": r.Method, "url": r.URL.String(), "headers": reqHdrs}})

	resp, err := t.base.RoundTrip(r)
	if err != nil {
		t.emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "error", Stage: "request_error", TraceID: t.traceID, Payload: map[string]interface{}{"error": err.Error()}})
		return nil, err
	}

	// emit response_headers for this hop
	respHdrs := copyHeaders(resp.Header)
	if t.redact {
		sanitizeHeaders(respHdrs, false)
	}
	t.emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: "http", EventType: "lifecycle", Stage: "response_headers", TraceID: t.traceID, Payload: map[string]interface{}{"status": resp.Status, "headers": respHdrs}})

	return resp, nil
}

func copyHeaders(h http.Header) map[string][]string {
	out := make(map[string][]string, len(h))
	for k, v := range h {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// sanitizeHeaders redacts sensitive header values. If req is true, redact Authorization/Cookie; for responses redact Set-Cookie.
func sanitizeHeaders(h map[string][]string, req bool) {
	for k := range h {
		lk := strings.ToLower(k)
		if req {
			if lk == "authorization" || lk == "cookie" {
				h[k] = []string{"REDACTED"}
			}
		} else {
			if lk == "set-cookie" {
				h[k] = []string{"REDACTED"}
			}
		}
	}
}
