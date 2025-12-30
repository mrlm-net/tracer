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

Example:

```bash
go run ./cmd/console -tracer http -data '{"ping":1}' -method POST https://example.com/api
```

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

All rights reserved © Martin Hrášek
[@mrlm-xyz](https://github.com/mrlm-xyz)
