FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o lettersmith ./cmd/server

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
