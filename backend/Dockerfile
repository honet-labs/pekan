FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/pekan-api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/pekan-worker ./cmd/worker/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/pekan-ai ./cmd/ai/main.go

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S pekan && adduser -S pekan -G pekan

WORKDIR /app

COPY --from=builder /app/pekan-api /app/pekan-api
COPY --from=builder /app/pekan-worker /app/pekan-worker
COPY --from=builder /app/pekan-ai /app/pekan-ai
COPY --from=builder /app/migrations /app/migrations

RUN chown -R pekan:pekan /app
USER pekan

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/healthz || exit 1
