package tracecommon

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mrlm-net/tracer/pkg/event"
)

// StartRequest emits a request_start lifecycle event and returns the traceID.
func StartRequest(ctx context.Context, emitter event.Emitter, protocol, addr string) string {
	traceID := uuid.NewString()
	emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: protocol, EventType: "lifecycle", Stage: "request_start", TraceID: traceID, Payload: map[string]interface{}{"addr": addr}})
	return traceID
}

// EmitDryRun emits a dry_run lifecycle event.
func EmitDryRun(ctx context.Context, emitter event.Emitter, protocol, traceID string) {
	emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: protocol, EventType: "lifecycle", Stage: "dry_run", TraceID: traceID})
}

// EmitError emits an error event for a specific stage.
func EmitError(ctx context.Context, emitter event.Emitter, protocol, stage, traceID string, err error) {
	if err == nil {
		return
	}
	emitter.Emit(ctx, event.Event{Timestamp: time.Now().UTC(), Protocol: protocol, EventType: "error", Stage: stage, TraceID: traceID, Payload: map[string]interface{}{"error": err.Error()}})
}

// BuildTags assembles ip_family, remote_ip and resolved_ips tags.
func BuildTags(chosenIP net.IP, resolved []net.IP, fam string) map[string]string {
	tags := map[string]string{}
	if fam != "" {
		tags["ip_family"] = fam
	}
	if chosenIP != nil {
		tags["remote_ip"] = chosenIP.String()
	}
	if len(resolved) > 0 {
		var sb strings.Builder
		for i, rip := range resolved {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(rip.String())
		}
		tags["resolved_ips"] = sb.String()
	}
	return tags
}

// EmitLifecycle emits a lifecycle event with optional connID, duration and payload.
func EmitLifecycle(ctx context.Context, emitter event.Emitter, protocol, stage, traceID, connID string, durationNS int64, tags map[string]string, payload map[string]interface{}) {
	e := event.Event{Timestamp: time.Now().UTC(), Protocol: protocol, EventType: "lifecycle", Stage: stage, TraceID: traceID}
	if connID != "" {
		e.ConnID = connID
	}
	if durationNS != 0 {
		e.DurationNS = durationNS
	}
	if len(tags) > 0 {
		e.Tags = tags
	}
	if payload != nil {
		e.Payload = payload
	}
	emitter.Emit(ctx, e)
}
