# syntax=docker/dockerfile:1.4
########################################
# Build stage: compile static binary
########################################
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache modules separately so changes to source don't bust module download cache
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source and build
COPY . .

# Build a static, optimized binary
# - CGO_ENABLED=0 for static linking
# - -trimpath to remove file paths
# - -ldflags "-s -w" to strip debug info
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /out/explorer ./cmd/explorer

########################################
# Final stage: minimal runtime
########################################
FROM alpine:3.19 AS runtime

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl tzdata && \
    addgroup -g 1000 explorer && \
    adduser -D -u 1000 -G explorer explorer && \
    mkdir -p /data /logs && \
    chown -R explorer:explorer /data /logs

# Copy CA certs and timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary
COPY --from=builder /out/explorer /usr/local/bin/explorer

# Switch to non-root user
USER explorer

WORKDIR /data

# Expose REST API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["explorer"]
