package console

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type headerFlags []string

func (h *headerFlags) String() string { return strings.Join(*h, ", ") }

func (h *headerFlags) Set(v string) error {
	*h = append(*h, v)
	return nil
}

// consoleConfig holds parsed CLI options for console package
type consoleConfig struct {
	Tracer            string
	DryRun            bool
	InjectTraceHeader bool
	Method            string
	Data              string
	PreferIP          string
	Output            string
	OutFile           string
	HeaderFlags       headerFlags
	Target            string
	// Redaction controls
	Redact          bool
	RedactRequests  bool
	RedactResponses bool
}

// parseFlags parses CLI args and returns a consoleConfig or error.
func parseFlags(args []string, stdout, stderr *os.File) (consoleConfig, error) {
	fs := flag.NewFlagSet("console", flag.ContinueOnError)
	fs.SetOutput(stderr)

	tracerFlag := fs.String("tracer", "http", "Type of tracer to use: udp, tcp, http, noop")
	dryRun := fs.Bool("dry-run", false, "If true, don't perform network requests; only show what would run")
	injectTraceHeader := fs.Bool("inject-trace-id", false, "If true, add X-Trace-Id header to outgoing requests")
	methodFlag := fs.String("method", "GET", "HTTP method to use for http tracer")
	dataFlag := fs.String("data", "", "Request body to send (for POST/PUT/PATCH)")
	preferIP := fs.String("prefer-ip", "", "IP preference: v4|v6|auto (default: auto)")
	outputFlagShort := fs.String("o", "json", "output format: json|html")
	outputFlag := fs.String("output", "json", "output format: json|html")
	outFileFlag := fs.String("out-file", "./tracer-report.html", "output path when using html")

	// redaction flags (default: enabled)
	redactFlag := fs.Bool("redact", true, "If true, redact sensitive headers in emitted events (Authorization, Cookie, Set-Cookie)")
	redactReqFlag := fs.Bool("redact-requests", true, "Redact request headers (Authorization, Cookie)")
	redactRespFlag := fs.Bool("redact-responses", true, "Redact response headers (Set-Cookie)")

	var header headerFlags
	fs.Var(&header, "H", "HTTP header (Name: value)")
	fs.Var(&header, "header", "HTTP header (Name: value)")

	if err := fs.Parse(args); err != nil {
		return consoleConfig{}, err
	}

	outputChoice := *outputFlag
	if outputChoice == "json" && *outputFlagShort != "json" {
		outputChoice = *outputFlagShort
	}

	flagArgs := fs.Args()
	if len(flagArgs) == 0 {
		prog := filepath.Base(os.Args[0])
		fmt.Fprintf(stderr, "Usage: %s [flags] target\n\n", prog)
		fs.PrintDefaults()
		return consoleConfig{}, fmt.Errorf("missing target")
	}

	cfg := consoleConfig{
		Tracer:            *tracerFlag,
		DryRun:            *dryRun,
		InjectTraceHeader: *injectTraceHeader,
		Method:            *methodFlag,
		Data:              *dataFlag,
		PreferIP:          *preferIP,
		Output:            outputChoice,
		OutFile:           *outFileFlag,
		HeaderFlags:       header,
		Target:            flagArgs[0],
		Redact:            *redactFlag,
		RedactRequests:    *redactReqFlag,
		RedactResponses:   *redactRespFlag,
	}
	return cfg, nil
}
