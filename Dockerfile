FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# Copy all source files
COPY . .

# Let Go handle dependency management automatically
RUN go mod tidy
RUN go mod download
RUN go mod verify

ENV GO111MODULE=on

RUN go build -o lettersmith ./cmd/server

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
