#FROM golang:tip-alpine3.22 AS backend

#WORKDIR /app

#COPY . .

#RUN go mod download
#RUN go mod tidy
#RUN go build -o bin/appsrv ./main.go

FROM golang:tip-alpine3.22 AS runtime

ENV GIN_MODE=release

WORKDIR /app

COPY --from=backend /app/bin/appsrv ./appsrv
COPY --from=backend /app/templates ./templates

CMD ["./appsrv"]

EXPOSE 8765