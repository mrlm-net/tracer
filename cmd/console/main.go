package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	eventpkg "github.com/mrlm-net/tracer/pkg/event"
	httppkg "github.com/mrlm-net/tracer/pkg/http"
	tcpkg "github.com/mrlm-net/tracer/pkg/tcp"
	udppkg "github.com/mrlm-net/tracer/pkg/udp"
)

var tracerFlag = flag.String("tracer", "http", "Type of tracer to use: udp, tcp, http, noop")
var dryRun = flag.Bool("dry-run", false, "If true, don't perform network requests; only show what would run")
var injectTraceHeader = flag.Bool("inject-trace-id", false, "If true, add X-Trace-Id header to outgoing requests")
var methodFlag = flag.String("method", "GET", "HTTP method to use for http tracer")
var dataFlag = flag.String("data", "", "Request body to send (for POST/PUT/PATCH)")
var preferIP = flag.String("prefer-ip", "", "IP preference: v4|v6|auto (default: auto)")
var outputFlagShort = flag.String("o", "json", "output format: json|html")
var outputFlag = flag.String("output", "json", "output format: json|html")
var outFileFlag = flag.String("out-file", "./tracer-report.html", "output path when using html")

type headerFlags []string

func (h *headerFlags) String() string { return strings.Join(*h, ", ") }

func (h *headerFlags) Set(v string) error {
	*h = append(*h, v)
	return nil
}

var headerFlag headerFlags

func main() {
	flag.Parse()
	// decide output mode (prefer short flag if provided)
	outputChoice := *outputFlag
	if outputChoice == "json" && *outputFlagShort != "json" {
		outputChoice = *outputFlagShort
	}
	flagArgs := flag.Args()
	if len(flagArgs) == 0 {
		prog := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] target\n\n", prog)
		flag.PrintDefaults()
		os.Exit(2)
	}
	// first non-flag argument is the target (host:port or URL)
	targetURL := flagArgs[0]

	switch *tracerFlag {
	case "udp":
		// ensure target is host:port for UDP
		addr := targetURL
		if strings.Contains(targetURL, "://") {
			u, err := url.Parse(targetURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid target %q: %v\n", targetURL, err)
				os.Exit(1)
			}
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				switch u.Scheme {
				case "http":
					port = "80"
				case "https":
					port = "443"
				default:
					fmt.Fprintf(os.Stderr, "no port in target %q and unknown scheme %q\n", targetURL, u.Scheme)
					os.Exit(1)
				}
			}
			addr = net.JoinHostPort(host, port)
		} else if !strings.Contains(targetURL, ":") {
			fmt.Fprintf(os.Stderr, "udp tracer target must be host:port or a URL with scheme\n")
			os.Exit(1)
		}

		var be *eventpkg.BufferingEmitter
		var emitter eventpkg.Emitter
		if outputChoice == "html" {
			be = eventpkg.NewBufferingEmitter()
			emitter = be
		} else {
			emitter = eventpkg.NewStdoutEmitter(os.Stdout, true, true)
		}
		opts := []udppkg.Option{udppkg.WithEmitter(emitter), udppkg.WithDryRun(*dryRun), udppkg.WithIPPreference(*preferIP)}
		if *dataFlag != "" {
			opts = append(opts, udppkg.WithDataString(*dataFlag))
		}
		if err := udppkg.TraceAddr(context.Background(), addr, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "udp tracer failed: %v\n", err)
			os.Exit(1)
		}
		if be != nil {
			events := be.Events()
			jb, err := json.Marshal(events)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to marshal events: %v\n", err)
				os.Exit(1)
			}
			tplBytes, err := os.ReadFile("./public/report.html")
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to read template: %v\n", err)
				os.Exit(1)
			}
			tplStr := string(tplBytes)
			// prefer replacing <!--DATA--> inside template if present
			if strings.Contains(tplStr, "<!--DATA-->") {
				tplStr = strings.Replace(tplStr, "<!--DATA-->", string(jb), 1)
			} else {
				script := fmt.Sprintf("<script id=\"__DATA__\" type=\"application/json\">%s</script>", jb)
				if strings.Contains(tplStr, "</body>") {
					tplStr = strings.Replace(tplStr, "</body>", script+"</body>", 1)
				} else {
					tplStr = tplStr + script
				}
			}
			outPath := *outFileFlag
			if err := os.WriteFile(outPath, []byte(tplStr), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write html: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stdout, "Wrote HTML report to "+outPath)
		}
	case "tcp":
		// ensure target is host:port for TCP
		addr := targetURL
		if strings.Contains(targetURL, "://") {
			u, err := url.Parse(targetURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid target %q: %v\n", targetURL, err)
				os.Exit(1)
			}
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				switch u.Scheme {
				case "http":
					port = "80"
				case "https":
					port = "443"
				default:
					fmt.Fprintf(os.Stderr, "no port in target %q and unknown scheme %q\n", targetURL, u.Scheme)
					os.Exit(1)
				}
			}
			addr = net.JoinHostPort(host, port)
		} else if !strings.Contains(targetURL, ":") {
			fmt.Fprintf(os.Stderr, "tcp tracer target must be host:port or a URL with scheme\n")
			os.Exit(1)
		}

		var be *eventpkg.BufferingEmitter
		var emitter eventpkg.Emitter
		if outputChoice == "html" {
			be = eventpkg.NewBufferingEmitter()
			emitter = be
		} else {
			emitter = eventpkg.NewStdoutEmitter(os.Stdout, true, true)
		}
		opts := []tcpkg.Option{tcpkg.WithEmitter(emitter), tcpkg.WithDryRun(*dryRun), tcpkg.WithIPPreference(*preferIP)}
		if *dataFlag != "" {
			opts = append(opts, tcpkg.WithDataString(*dataFlag))
		}
		if err := tcpkg.TraceAddr(context.Background(), addr, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "tcp tracer failed: %v\n", err)
			os.Exit(1)
		}
		if be != nil {
			events := be.Events()
			jb, err := json.Marshal(events)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to marshal events: %v\n", err)
				os.Exit(1)
			}
			tplBytes, err := os.ReadFile("./public/report.html")
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to read template: %v\n", err)
				os.Exit(1)
			}
			tplStr := string(tplBytes)
			if strings.Contains(tplStr, "<!--DATA-->") {
				tplStr = strings.Replace(tplStr, "<!--DATA-->", string(jb), 1)
			} else {
				script := fmt.Sprintf("<script id=\"__DATA__\" type=\"application/json\">%s</script>", jb)
				if strings.Contains(tplStr, "</body>") {
					tplStr = strings.Replace(tplStr, "</body>", script+"</body>", 1)
				} else {
					tplStr = tplStr + script
				}
			}
			outPath := *outFileFlag
			if err := os.WriteFile(outPath, []byte(tplStr), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write html: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stdout, "Wrote HTML report to "+outPath)
		}
	case "http":
		var be *eventpkg.BufferingEmitter
		var emitter eventpkg.Emitter
		if outputChoice == "html" {
			be = eventpkg.NewBufferingEmitter()
			emitter = be
		} else {
			emitter = eventpkg.NewStdoutEmitter(os.Stdout, true, true)
		}
		opts := []httppkg.Option{httppkg.WithEmitter(emitter), httppkg.WithDryRun(*dryRun), httppkg.WithInjectTraceHeader(*injectTraceHeader), httppkg.WithIPPreference(*preferIP)}
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
		if be != nil {
			events := be.Events()
			jb, err := json.Marshal(events)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to marshal events: %v\n", err)
				os.Exit(1)
			}
			tplBytes, err := os.ReadFile("./public/report.html")
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to read template: %v\n", err)
				os.Exit(1)
			}
			tplStr := string(tplBytes)
			if strings.Contains(tplStr, "<!--DATA-->") {
				tplStr = strings.Replace(tplStr, "<!--DATA-->", string(jb), 1)
			} else {
				script := fmt.Sprintf("<script id=\"__DATA__\" type=\"application/json\">%s</script>", jb)
				if strings.Contains(tplStr, "</body>") {
					tplStr = strings.Replace(tplStr, "</body>", script+"</body>", 1)
				} else {
					tplStr = tplStr + script
				}
			}
			outPath := *outFileFlag
			if err := os.WriteFile(outPath, []byte(tplStr), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write html: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stdout, "Wrote HTML report to "+outPath)
		}
	default:
		panic("Unknown tracer type: " + *tracerFlag)
	}
}

// HTTP tracing implemented in pkg/http
