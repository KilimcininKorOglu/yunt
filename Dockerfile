# =============================================================================
# Yunt Mail Server - Multi-stage Dockerfile
# =============================================================================
# This Dockerfile creates an optimized, minimal container image for Yunt.
# It uses multi-stage builds to separate build dependencies from runtime.
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Web UI Builder
# Build the SvelteKit frontend
# -----------------------------------------------------------------------------
FROM node:22-alpine AS web-builder

WORKDIR /app

# Copy web package files first for layer caching
COPY web/package.json web/package-lock.json ./

# Install dependencies
RUN npm ci --no-audit --no-fund

# Copy web source files
COPY web/ ./

# Build the web UI
# SvelteKit outputs to ../webui/dist (relative to web directory)
# We'll output to /webui/dist in the container
RUN mkdir -p /webui/dist && \
    npm run build -- --mode production

# -----------------------------------------------------------------------------
# Stage 2: Go Builder
# Compile the Go binary with embedded Web UI
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
# git: for go mod operations
# ca-certificates, tzdata: for TLS and timezone support
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy go module files first for layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy the web UI build output from previous stage
# SvelteKit adapter-static outputs to /webui/dist (see svelte.config.js)
COPY --from=web-builder /webui/dist/ /app/webui/dist/

# Copy Go source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY docs/ ./docs/
COPY webui/embed.go ./webui/

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary with optimizations
# -s: omit symbol table
# -w: omit DWARF debugging info
# CGO_ENABLED=0: pure-Go build (modernc.org/sqlite, no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \
    -trimpath \
    -o /app/yunt \
    ./cmd/yunt

# -----------------------------------------------------------------------------
# Stage 3: Runtime
# Minimal Alpine image with only the compiled binary
# -----------------------------------------------------------------------------
FROM alpine:3.21 AS runtime

# Labels for container metadata
LABEL org.opencontainers.image.title="Yunt Mail Server" \
      org.opencontainers.image.description="Lightweight development mail server" \
      org.opencontainers.image.vendor="Yunt" \
      org.opencontainers.image.source="https://github.com/yunt/yunt"

# Install minimal runtime dependencies
# ca-certificates: for HTTPS/TLS connections
# tzdata: for timezone support
# curl: for health checks (smaller than wget on alpine)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user and group
RUN addgroup -g 1000 -S yunt && \
    adduser -u 1000 -S -G yunt -h /home/yunt -s /sbin/nologin yunt

# Create directories for data and configuration
RUN mkdir -p /var/lib/yunt /etc/yunt && \
    chown -R yunt:yunt /var/lib/yunt /etc/yunt

# Copy the binary from builder stage
COPY --from=go-builder /app/yunt /usr/local/bin/yunt

# Ensure binary is executable
RUN chmod +x /usr/local/bin/yunt

# Switch to non-root user
USER yunt

# Set working directory
WORKDIR /var/lib/yunt

# Expose ports
# 1025: SMTP
# 1143: IMAP
# 8025: Web UI / API
EXPOSE 1025 1143 8025

# Define volume for persistent data
VOLUME ["/var/lib/yunt"]

# Environment variables with sensible defaults
ENV YUNT_DATABASE_DSN=/var/lib/yunt/yunt.db \
    YUNT_LOGGING_OUTPUT=stdout \
    YUNT_LOGGING_FORMAT=json

# Health check
# Check if all services are ready (database, SMTP, IMAP)
# Uses /ready endpoint which verifies all enabled services are running
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl --fail --silent http://localhost:8025/ready || exit 1

# Default command
ENTRYPOINT ["/usr/local/bin/yunt"]
CMD ["serve"]
