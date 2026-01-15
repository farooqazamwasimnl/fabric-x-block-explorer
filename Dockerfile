# syntax=docker/dockerfile:1.4
########################################
# Build stage: compile static binary
########################################
FROM golang:1.25.4 AS builder

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
    go build -trimpath -ldflags="-s -w" -o /out/explorer ./cmd/explorer

########################################
# Final stage: minimal runtime
########################################
FROM scratch AS runtime

# Copy CA certs from builder so TLS works (if needed)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the compiled binary
COPY --from=builder /out/explorer /explorer

# Use a non-root user if you add one; scratch has no passwd, so run as root here.
EXPOSE 8080

ENTRYPOINT ["/explorer"]
