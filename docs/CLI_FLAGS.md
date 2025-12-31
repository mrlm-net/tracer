# CLI Flags Reference

This document is a comprehensive reference for the `cmd/console` CLI flags.

## Common flags

- `-tracer` : `http` (default), `tcp`, `udp`, `noop`.
- `-dry-run` : If true, emit lifecycle events but do not perform network I/O.
- `-inject-trace-id` : For HTTP, add `X-Trace-Id` header to outgoing requests.
- `-method` : HTTP method to use (GET/POST/PUT/...).
- `-data` : Request body to send for HTTP/TCP/UDP.
- `-H` / `-header` : Repeatable header flag in the format `Name: value`.

## Output flags

- `-o`, `-output` : `json` (default) or `html`.
- `--out-file` : Path to write the HTML report when `-o html` is selected.

## Redaction flags

- `--redact` (default: `true`): Coarse-grained toggle to enable/disable redaction.
- `--redact-requests` (default: `true`): Redact Authorization/Cookie on requests.
- `--redact-responses` (default: `true`): Redact Set-Cookie on responses.

## IP resolution

- `-prefer-ip` : IP preference when resolving hostnames. Accepts `v4`, `v6`, or `auto` (default).

## Examples

```bash
# Basic HTTP trace
tracer -tracer http https://example.com/

# Debug without redaction (unsafe)
tracer -tracer http --redact=false https://example.com/

# Write HTML report
tracer -tracer http -o html --out-file ./report.html https://example.com/
```
