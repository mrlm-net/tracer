package console

import (
	"context"
	"fmt"
	"net/http"
	"os"

	httppkg "github.com/mrlm-net/tracer/pkg/http"
	tcpkg "github.com/mrlm-net/tracer/pkg/tcp"
	udppkg "github.com/mrlm-net/tracer/pkg/udp"
)

// dispatchTrace runs the appropriate tracer based on cfg and returns an exit code.
func dispatchTrace(ctx context.Context, cfg consoleConfig, stdout, stderr *os.File) int {
	switch cfg.Tracer {
	case "udp":
		addr, err := targetToAddr(cfg.Target, "udp")
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}
		emitter, be := makeEmitter(cfg.Output, stdout)
		opts := []udppkg.Option{udppkg.WithEmitter(emitter), udppkg.WithDryRun(cfg.DryRun), udppkg.WithIPPreference(cfg.PreferIP)}
		if cfg.Data != "" {
			opts = append(opts, udppkg.WithDataString(cfg.Data))
		}
		if err := udppkg.TraceAddr(ctx, addr, opts...); err != nil {
			fmt.Fprintf(stderr, "udp tracer failed: %v\n", err)
			return 1
		}
		if be != nil {
			if err := writeHTMLReport(cfg.OutFile, be.Events(), stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
				return 1
			}
		}
		return 0
	case "tcp":
		addr, err := targetToAddr(cfg.Target, "tcp")
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}
		emitter, be := makeEmitter(cfg.Output, stdout)
		opts := []tcpkg.Option{tcpkg.WithEmitter(emitter), tcpkg.WithDryRun(cfg.DryRun), tcpkg.WithIPPreference(cfg.PreferIP)}
		if cfg.Data != "" {
			opts = append(opts, tcpkg.WithDataString(cfg.Data))
		}
		if err := tcpkg.TraceAddr(ctx, addr, opts...); err != nil {
			fmt.Fprintf(stderr, "tcp tracer failed: %v\n", err)
			return 1
		}
		if be != nil {
			if err := writeHTMLReport(cfg.OutFile, be.Events(), stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
				return 1
			}
		}
		return 0
	case "http":
		emitter, be := makeEmitter(cfg.Output, stdout)
		opts := []httppkg.Option{httppkg.WithEmitter(emitter), httppkg.WithDryRun(cfg.DryRun), httppkg.WithInjectTraceHeader(cfg.InjectTraceHeader), httppkg.WithIPPreference(cfg.PreferIP)}
		if cfg.Method != "" && cfg.Method != "GET" {
			opts = append(opts, httppkg.WithMethod(cfg.Method))
		}
		if cfg.Data != "" {
			opts = append(opts, httppkg.WithBodyString(cfg.Data))
			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			opts = append(opts, httppkg.WithHeaders(h))
		}
		if len(cfg.HeaderFlags) > 0 {
			h := make(http.Header)
			for _, hv := range cfg.HeaderFlags {
				parts := splitHeader(hv)
				if parts == nil {
					fmt.Fprintf(stderr, "invalid header %q, expected 'Name: value'\n", hv)
					return 2
				}
				h.Add(parts[0], parts[1])
			}
			opts = append(opts, httppkg.WithHeaders(h))
		}
		// Wire redaction options from CLI to the http tracer. Apply coarse-grained
		// option first then fine-grained options so specific flags override.
		opts = append(opts, httppkg.WithRedact(cfg.Redact), httppkg.WithRedactRequests(cfg.RedactRequests), httppkg.WithRedactResponses(cfg.RedactResponses))

		if err := httppkg.TraceURL(ctx, cfg.Target, opts...); err != nil {
			fmt.Fprintf(stderr, "http tracer failed: %v\n", err)
			return 1
		}
		if be != nil {
			if err := writeHTMLReport(cfg.OutFile, be.Events(), stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
				return 1
			}
		}
		return 0
	default:
		fmt.Fprintf(stderr, "Unknown tracer type: %s\n", cfg.Tracer)
		return 1
	}
}

// splitHeader parses "Name: value" into [name, value] or returns nil.
func splitHeader(hv string) []string {
	// local helper so we avoid importing strings twice elsewhere

	for range 2 {
		// placeholder; actual parsing done below
	}
	// simple split N=2
	for i, p := range []string{"", ""} {
		_ = i
		_ = p
	}
	// implement proper split
	// (do not import extra packages here; reuse strings via split in caller scope if needed)
	// but we can implement quickly:
	s := hv
	idx := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	name := stringsTrimSpace(s[:idx])
	value := stringsTrimSpace(s[idx+1:])
	if name == "" {
		return nil
	}
	return []string{name, value}
}

// small local helpers to avoid extra imports in the top of this file
func stringsTrimSpace(s string) string {
	// replicate strings.TrimSpace minimal
	start := 0
	end := len(s)
	for start < end {
		c := s[start]
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\f' || c == '\v' {
			start++
			continue
		}
		break
	}
	for end > start {
		c := s[end-1]
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\f' || c == '\v' {
			end--
			continue
		}
		break
	}
	return s[start:end]
}
