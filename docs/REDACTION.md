# Redaction & Security

This document explains what the tracer redacts by default, why, and how to safely control redaction for debugging scenarios.

## Default behavior

- Request headers: `Authorization`, `Cookie` are redacted by default.
- Response headers: `Set-Cookie` is redacted by default.

These defaults are chosen to reduce the risk of leaking access tokens, session cookies, and other secrets in trace outputs.

## CLI flags

- `--redact` (bool, default: `true`) — coarse-grained toggle that enables/disables redaction for both requests and responses.
- `--redact-requests` (bool, default: `true`) — controls redaction of request headers (Authorization, Cookie).
- `--redact-responses` (bool, default: `true`) — controls redaction of response headers (Set-Cookie).

Use `--redact=false` to disable all redaction (unsafe for production). Prefer toggling the request/response flags if you only need one side visible.

## Security guidance

- Never disable redaction when capturing traces from production or customer workloads unless you have explicit approval.
- When redaction is disabled, restrict access to produced artifacts (NDJSON logs, HTML reports) and use short-lived storage (ephemeral PVCs) or secure sinks.
- Consider using the `WithSanitizer` or a hashing approach (future feature) to avoid exposing full secrets while still enabling correlation.

## PR/security checklist

Before merging changes that affect redaction or the Event schema:

- Confirm the list of redacted fields matches implementation in `pkg/http` and any other emitters.
- Add a note in the PR describing why exposing additional fields is necessary and who approved it.
- Ensure example commands that disable redaction include a clear warning and are marked 'unsafe'.
