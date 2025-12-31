# Emitters and Output Formats

This document explains the emitters used by the tracer and the available output formats.

## StdoutEmitter (default)

- Streams NDJSON event objects to stdout.
- Prints a short human summary line for each event to make console runs easier to read.

## BufferingEmitter + HTML Report

- When `-o html` is selected the CLI uses a `BufferingEmitter` which collects events in memory and writes them into the HTML template (`public/report.html`).
- The HTML report is a self-contained interactive viewer which embeds the event JSON and renders timelines and header details.

## Event schema

The `Event` type is defined in `pkg/event`. Events include fields such as `Timestamp`, `Protocol`, `EventType`, `Stage`, `TraceID`, `DurationNS`, and `Payload` (map). Review `pkg/event` for the canonical structure and stable fields.

## Memory considerations

- The HTML report collects all events in memory; avoid using `-o html` for very large traces or high-throughput sampling. Prefer NDJSON streaming for long-running or high-volume captures.
