````markdown
# Containers & K8s (Platform Debugging)

Guidance for running the tracer in containers and Kubernetes (K8s) for platform engineers.

## Docker image example

Create a minimal image that contains the `tracer` binary and run it to produce an HTML report:

```Dockerfile
FROM golang:1.21-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/tracer ./cmd/console

FROM scratch
COPY --from=build /bin/tracer /tracer
ENTRYPOINT ["/tracer"]
```

Run and write report to host:

```bash
docker run --rm -v $(pwd):/out mrlm/tracer:latest /tracer -tracer http -o html --out-file /out/report.html https://example.com/
```

## Kubernetes / K8s

Suggested ephemeral debug Job (ensure RBAC and policies allow it):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: tracer-debug
spec:
  template:
    spec:
      containers:
      - name: tracer
        image: mrlm/tracer:latest
        command: ["/tracer", "-tracer", "http", "https://example.com/"]
      restartPolicy: Never
  backoffLimit: 0
```

Collect output via `kubectl logs` or mount a PVC and write `--out-file` to the volume. Always redact sensitive headers when capturing from shared or customer workloads.

## Security checklist for cluster debugging

- Confirm who is authorized to run debugging jobs.
- Prefer `--redact` defaults; only disable redaction with explicit approval.
- Use ephemeral storage and remove artifacts after debugging.

````
