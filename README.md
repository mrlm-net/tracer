# mrlm-net/tracer

Lightweight network tracer for HTTP/TCP/UDP that emits normalized lifecycle events (NDJSON + human summary) for debugging and instrumentation.

|  |  |
|--|--|
| Package name | github.com/mrlm-net/tracer |
| Go version | 1.21+ |
| License | Apache-2.0 |
| Platform | Cross-platform |

## Table of contents

- [Installation](#installation)
- [Usage](#usage)
- [CLI](#cli)
- [Packages / API](#packages--api)
- [Examples](#examples)
- [Debugging](#debugging)
- [Contributing](#contributing)

## Installation

This project uses Go modules. Add it to your module or install the CLI with:

```bash
go get github.com/mrlm-net/tracer
```

Remote install (recommended for CLI users):

```bash
go install github.com/mrlm-net/tracer/cmd/console@latest
```

The installed binary will be placed in `$(go env GOBIN)` or `$(go env GOPATH)/bin` — add that to your `PATH` if needed.

## Requirements

- Go 1.21 or higher

## Usage

The repository includes a small CLI in `cmd/console` that demonstrates the tracer behavior.

Run the console tracer (default `http`):

```bash
go run ./cmd/console -tracer http https://example.com/
```

Trace TCP (host:port or URL):

```bash
go run ./cmd/console -tracer tcp example.com:443
# or using a URL (port inferred for http/https)
go run ./cmd/console -tracer tcp https://example.com/
```

Trace UDP:

```bash
go run ./cmd/console -tracer udp 127.0.0.1:9999
```

## CLI

Important flags (see `cmd/console/main.go`):

- `-tracer` : `http` (default), `tcp`, `udp`, `noop`
- `-dry-run` : If true, emit lifecycle events but do not perform network I/O
- `-inject-trace-id` : For HTTP, add `X-Trace-Id` header to outgoing requests
- `-method` : HTTP method for `http` tracer (GET/POST/PUT/...)
- `-data` : Request payload to send for TCP/UDP or HTTP body
- `-H` : Repeatable header flags for HTTP (format `Name: value`)

- `-prefer-ip` : IP preference when resolving hostnames. Accepts `v4`, `v6`, or `auto` (default). When an IP literal is provided (e.g. `127.0.0.1` or `[::1]`) the tracer will honor the literal family.

- `-o` / `-output` : `json` (default) or `html`. When set to `html` the CLI collects all events and writes a single HTML report instead of streaming NDJSON to stdout.
- `--out-file` : Path to write the HTML report when `-o html` is selected (default `./tracer-report.html`).

Example:

```bash
go run ./cmd/console -tracer http -data '{"ping":1}' -method POST https://example.com/api
```

Example: generate HTML report

```bash
go run ./cmd/console -o html --out-file ./report.html -tracer http https://example.com/
```

## Quick Start

Get a runnable binary and run a simple HTTP trace (redaction enabled by default):

```bash
go install github.com/mrlm-net/tracer/cmd/console@latest
tracer -tracer http https://example.com/
```

The CLI streams NDJSON events to stdout by default and redacts sensitive headers (Authorization, Cookie, Set-Cookie) unless you explicitly disable redaction.

## Redaction & Security

By default the tracer redacts sensitive header values from emitted events to avoid leaking secrets. The default behavior removes values for `Authorization`, `Cookie` (requests) and `Set-Cookie` (responses). You can control this behavior with the CLI flags `--redact`, `--redact-requests`, and `--redact-responses` (see docs/REDACTION.md for details and security guidance).

Important: disabling redaction (`--redact=false`) may expose secrets. Only disable it for local debugging or when you have explicit approval to capture sensitive data.

## Emitters & Output Formats

The tracer supports multiple emitter behaviors:

- `StdoutEmitter` (default): streams NDJSON events to stdout and prints a short human summary line for each event.
- `BufferingEmitter`: collects events in memory and writes a single HTML report when the trace completes (use `-o html` and `--out-file`).

See docs/EMITTERS_AND_OUTPUTS.md for the event schema and details about emitters and memory considerations.

## Running in Containers

You can run the tracer inside a container for portable debugging. Build an image containing the `tracer` binary and run it with appropriate mounts to capture reports:

```bash
# build and run (example)
docker build -t mrlm/tracer:latest .
docker run --rm -v $(pwd):/out mrlm/tracer:latest /tracer -tracer http -o html --out-file /out/report.html https://example.com/
```

When running in multi-tenant environments (Kubernetes/K8s), follow the guidance in docs/CONTAINERS_AND_K8S.md to avoid accidental data leakage and to handle RBAC and network policies correctly.

## K8s / Platform Debugging

Platform engineers can use the tracer as an ephemeral debugging tool in K8s clusters. Recommended approaches include running the tracer as a short-lived Job or `kubectl run` pod and collecting the HTML report via a PVC or `kubectl cp`. Always follow organizational security policies when collecting traces from customer workloads — prefer redaction and secure sinks. See docs/CONTAINERS_AND_K8S.md for example manifests and runbooks.

## Configuration

Currently the tracer is configured via CLI flags. See docs/CONFIGURATION.md for recommendations on env var mappings and wrapper scripts for CI or platform automation.

## Packages / API

- `pkg/event` — normalized `Event` type and `Emitter` interface; `NewStdoutEmitter` prints NDJSON + pretty summary.
- `pkg/http` — HTTP tracer; `TraceURL(ctx, url, opts...)` with functional options: `WithEmitter`, `WithDryRun`, `WithInjectTraceHeader`, `WithMethod`, `WithBodyString`, `WithHeaders`, etc.
- `pkg/tcp` — TCP tracer; `TraceAddr(ctx, addr, opts...)` with `WithEmitter`, `WithDryRun`, `WithDataString`, `WithTimeout`.
- `pkg/udp` — UDP tracer; `TraceAddr(ctx, addr, opts...)` with `WithEmitter`, `WithDryRun`, `WithDataString`, `WithTimeout`, `WithRecvBuffer`.

These packages follow the functional `Option` pattern used in `pkg/http` so they are easy to compose from code or the CLI.

## Examples

See `cmd/console` for a minimal example that instantiates a `StdoutEmitter` and calls the relevant tracer with options.

To write code against the tracer packages:

```go
em := event.NewStdoutEmitter(os.Stdout, true, true)
// HTTP
http.TraceURL(ctx, "https://example.com/", http.WithEmitter(em), http.WithDryRun(false))
// TCP
tcp.TraceAddr(ctx, "example.com:443", tcp.WithEmitter(em), tcp.WithDataString("hello"))

IPv4 / IPv6 examples (CLI):

```bash
# IPv4 literal
go run ./cmd/console -tracer tcp 127.0.0.1:8080

# IPv6 literal (note brackets when specifying a port)
go run ./cmd/console -tracer tcp [::1]:8080

# Link-local IPv6 with zone (example):
go run ./cmd/console -tracer udp "fe80::1%en0:9999"

# Prefer IPv6 when resolving a hostname
go run ./cmd/console -tracer http -prefer-ip v6 https://example.com/
```

Target-argument examples (copy & run):

```bash
# HTTP (URL target)
go run ./cmd/console -tracer http https://example.com/

# TCP (host:port target)
go run ./cmd/console -tracer tcp example.com:443

# UDP (host:port target)
go run ./cmd/console -tracer udp 127.0.0.1:9999
```

## Debugging

Enable verbose inspection by using the `StdoutEmitter` (default in CLI). The emitter prints a short human-friendly summary line and a JSON event line for each emitted event.

Common issues:

- Network reachability: ensure host/port are reachable from the machine running the tracer.
- For TCP/UDP, prefer `host:port` targets; the console accepts URLs and will infer `80`/`443` for `http`/`https`.

## Contributing

Contributions welcome. Please follow the repository contribution guidelines in the organization if present.

## License

Apache-2.0

---

All rights reserved © Martin Hrášek
[@mrlm-xyz](https://github.com/mrlm-xyz)
