# Stage 1: Build Binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically-linked binary without CGO dependency
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o trading-bot main.go

# Stage 2: Production Minimal Image
FROM alpine:latest

WORKDIR /app

# Install Root CA certificates for SSL/TLS external API connections & timezone data
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder stage
COPY --from=builder /app/trading-bot .

# Set default timezone to Asia/Jakarta (WIB)
ENV TZ=Asia/Jakarta

ENTRYPOINT ["./trading-bot"]
