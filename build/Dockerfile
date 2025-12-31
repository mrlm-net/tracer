FROM golang:1.21-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/tracer ./cmd/console

FROM scratch
COPY --from=build /bin/tracer /tracer
ENTRYPOINT ["/tracer"]