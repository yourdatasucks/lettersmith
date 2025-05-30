FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy all source files first
COPY . .

# Debug: See what's actually in the container in GitHub Actions
RUN echo "=== DEBUGGING IN GITHUB ACTIONS ==="
RUN ls -la
RUN echo "=== INTERNAL DIRECTORY ==="
RUN ls -la internal/
RUN echo "=== CONFIG DIRECTORY ==="
RUN ls -la internal/config/
RUN echo "=== GO.MOD CONTENT ==="
RUN cat go.mod
RUN echo "=== GO VERSION ==="
RUN go version
RUN echo "=== GO ENV ==="
RUN go env
RUN echo "=== CURRENT WORKING DIRECTORY ==="
RUN pwd
RUN echo "=== GOPATH AND GOMOD ==="
RUN echo "GOPATH: $GOPATH"
RUN echo "GO111MODULE: $GO111MODULE"

# Ensure Go modules are properly initialized
RUN go mod download
RUN go mod verify

ENV GO111MODULE=on

# Try building with verbose output
RUN go build -v -o lettersmith ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/lettersmith .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./lettersmith"]
