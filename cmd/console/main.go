package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	eventpkg "github.com/mrlm-net/tracer/pkg/event"
	httppkg "github.com/mrlm-net/tracer/pkg/http"
)

var tracerFlag = flag.String("tracer", "http", "Type of tracer to use: udp, tcp, http, noop")
var dryRun = flag.Bool("dry-run", false, "If true, don't perform network requests; only show what would run")
var injectTraceHeader = flag.Bool("inject-trace-id", false, "If true, add X-Trace-Id header to outgoing requests")

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
		if err := httppkg.TraceURL(context.Background(), targetURL, httppkg.WithEmitter(emitter), httppkg.WithDryRun(*dryRun), httppkg.WithInjectTraceHeader(*injectTraceHeader)); err != nil {
			fmt.Fprintf(os.Stderr, "http tracer failed: %v\n", err)
			os.Exit(1)
		}
	default:
		panic("Unknown tracer type: " + *tracerFlag)
	}
}

// HTTP tracing implemented in pkg/http
