FROM golang:1.23-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o lettersmith ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/lettersmith .
COPY --from=builder /build/web ./web
COPY --from=builder /build/migrations ./migrations

ARG USER_ID=1000
ARG GROUP_ID=1000

RUN addgroup -g ${GROUP_ID} appgroup && \
    adduser -D -u ${USER_ID} -G appgroup appuser

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./lettersmith"] 