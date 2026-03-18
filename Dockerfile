# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api ./cmd/api/main.go

# Build Worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /worker ./cmd/worker/main.go

# API image
FROM alpine:3.19 AS api

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /api /app/api

EXPOSE 8080

CMD ["/app/api"]

# Worker image
FROM alpine:3.19 AS worker

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata ffmpeg yt-dlp

COPY --from=builder /worker /app/worker

CMD ["/app/worker"]
