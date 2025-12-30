package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	eventpkg "github.com/mrlm-net/tracer/pkg/event"
	httppkg "github.com/mrlm-net/tracer/pkg/http"
)

var tracerFlag = flag.String("tracer", "http", "Type of tracer to use: udp, tcp, http, noop")
var dryRun = flag.Bool("dry-run", false, "If true, don't perform network requests; only show what would run")
var injectTraceHeader = flag.Bool("inject-trace-id", false, "If true, add X-Trace-Id header to outgoing requests")
var methodFlag = flag.String("method", "GET", "HTTP method to use for http tracer")
var dataFlag = flag.String("data", "", "Request body to send (for POST/PUT/PATCH)")
type headerFlags []string

func (h *headerFlags) String() string { return strings.Join(*h, ", ") }

func (h *headerFlags) Set(v string) error {
	*h = append(*h, v)
	return nil
}

var headerFlag headerFlags

func main() {
	flag.Parse()
	// Default target URL; can be overridden by first non-flag argument
	targetURL := "https://example.com/"
	if args := flag.Args(); len(args) > 0 {
		targetURL = args[0]
	}

	switch *tracerFlag {
	case "udp":
		fmt.Fprintln(os.Stderr, "udp tracer not implemented yet")
	case "tcp":
		fmt.Fprintln(os.Stderr, "tcp tracer not implemented yet")
	case "http":
		emitter := eventpkg.NewStdoutEmitter(os.Stdout, true, true)
		opts := []httppkg.Option{httppkg.WithEmitter(emitter), httppkg.WithDryRun(*dryRun), httppkg.WithInjectTraceHeader(*injectTraceHeader)}
		if *methodFlag != "" && *methodFlag != "GET" {
			opts = append(opts, httppkg.WithMethod(*methodFlag))
		}
		if *dataFlag != "" {
			opts = append(opts, httppkg.WithBodyString(*dataFlag))
			// set a default content-type when sending data
			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			opts = append(opts, httppkg.WithHeaders(h))
		}
		// parse -H/--header flags
		if len(headerFlag) > 0 {
			h := make(http.Header)
			for _, hv := range headerFlag {
				parts := strings.SplitN(hv, ":", 2)
				if len(parts) != 2 {
					fmt.Fprintf(os.Stderr, "invalid header %q, expected 'Name: value'\n", hv)
					os.Exit(2)
				}
				name := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				h.Add(name, value)
			}
			opts = append(opts, httppkg.WithHeaders(h))
		}
		if err := httppkg.TraceURL(context.Background(), targetURL, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "http tracer failed: %v\n", err)
			os.Exit(1)
		}
	default:
		panic("Unknown tracer type: " + *tracerFlag)
	}
}

// HTTP tracing implemented in pkg/http
