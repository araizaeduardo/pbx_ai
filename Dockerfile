# Multi-stage Dockerfile for PBX AI Communications Bridge

# Stage 1: Build binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make build-base

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile binary
RUN make pbx

# Stage 2: Runtime image
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies (ffmpeg & ffprobe for media transcoding, ca-certificates, tzdata)
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user and required folders
RUN addgroup -S pbxgroup && adduser -S pbxuser -G pbxgroup \
    && mkdir -p /app/media /app/cache /app/data \
    && chown -R pbxuser:pbxgroup /app

# Copy binary from builder
COPY --from=builder /app/build/pbx /app/pbx

# Set permissions
RUN chmod +x /app/pbx

USER pbxuser

# Expose Web/Dashboard/API and Asterisk AGI ports
EXPOSE 35248 4573

# Healthcheck monitoring endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://127.0.0.1:35248/health || exit 1

# Default execution
ENTRYPOINT ["/app/pbx"]
CMD ["--start", "--db-type", "sqlite", "--db-name", "/app/data/pbx.db", "--asterisk", "--asterisk-agi", "0.0.0.0:4573", "--chat", "--chat-audio", "--web-host", "0.0.0.0", "--web-port", "35248", "--media-path", "/app/media/", "--cache", "/app/cache/"]

