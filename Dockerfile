# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN go build -o homebridge-exporter .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/homebridge-exporter .

# Expose metrics port
EXPOSE 9123

# Environment variables for configuration
# HOMEBRIDGE_USERNAME - Homebridge username (required)
# HOMEBRIDGE_PASSWORD - Homebridge password (required)
# HOMEBRIDGE_URI - Homebridge API endpoint (default: http://localhost:8581)
# HOMEBRIDGE_PORT - Metrics server port (default: 9123)
# HOMEBRIDGE_PREFIX - Metric name prefix (default: homebridge)
# HOMEBRIDGE_KEYFILE - Authorization keys YAML file (default: authorization-keys.yml)
# HOMEBRIDGE_DEBUG - Enable debug logging (default: false)

ENTRYPOINT ["/app/homebridge-exporter"]
