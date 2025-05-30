FROM golang:1.23-alpine AS builder

# Match the module path exactly to satisfy internal imports
WORKDIR /go/src/github.com/yourdatasucks/lettersmith

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN cat go.mod

ENV GO111MODULE=on

RUN go build -o lettersmith ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /go/src/github.com/yourdatasucks/lettersmith/lettersmith .
COPY --from=builder /go/src/github.com/yourdatasucks/lettersmith/web ./web
COPY --from=builder /go/src/github.com/yourdatasucks/lettersmith/migrations ./migrations

RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./lettersmith"]
