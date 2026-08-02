FROM golang:1.25-alpine

WORKDIR /app

# Install Root CA certificates untuk koneksi SSL/TLS API eksternal & timezone data
RUN apk add --no-cache ca-certificates tzdata

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code & berkas pendukung
COPY . .

# Build binary aplikasi
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o trading-bot main.go

# Set default timezone ke Asia/Jakarta (WIB)
ENV TZ=Asia/Jakarta

ENTRYPOINT ["./trading-bot"]
