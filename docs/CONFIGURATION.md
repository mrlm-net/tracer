# Configuration

The tracer is currently configured via CLI flags. This document describes configuration options and recommendations for automation or CI usage.

## CLI-first configuration

- The primary configuration mechanism is CLI flags (see docs/CLI_FLAGS.md).
- Example wrapper script reads env vars and calls the binary with equivalent flags.

## Recommended env var mapping (optional)

If you add environment variable support, consider the following names:

- `TRACER_REDACT` — `true|false` (coarse-grained redaction)
- `TRACER_REDACT_REQUESTS` — `true|false`
- `TRACER_REDACT_RESPONSES` — `true|false`
- `TRACER_OUTPUT` — `json|html`

Document any env var support here and provide examples of wrapper scripts used by CI or platform automation.
