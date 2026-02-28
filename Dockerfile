# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Run stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g "" appuser

WORKDIR /app

COPY --from=builder /server .

USER appuser

EXPOSE 8080

# Config via env (e.g. PORT, SLIDING_WINDOW_SIZE, LOG_LEVEL, LOG_FORMAT)
ENTRYPOINT ["./server"]
