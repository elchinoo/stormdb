# StormDB v0.4-alpha Dockerfile
# Multi-stage build for StormDB v0.4-alpha with plugin support

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    gcc \
    musl-dev \
    binutils-gold \
    postgresql-client

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build plugins first (requires CGO for .so files)
RUN CGO_ENABLED=1 make plugins

# Build the main application
RUN CGO_ENABLED=0 GOOS=linux make build

# Verify builds
RUN ls -la build/ && ls -la build/plugins/

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    postgresql-client \
    curl \
    jq

# Create app user
RUN addgroup -g 1000 -S stormdb && \
    adduser -u 1000 -S stormdb -G stormdb

# Create necessary directories
RUN mkdir -p \
    /app/config \
    /app/plugins \
    /app/data \
    /app/logs \
    /var/lib/stormdb/plugins && \
    chown -R stormdb:stormdb /app /var/lib/stormdb

# Copy binary and plugins from builder
COPY --from=builder /app/build/stormdb /usr/local/bin/stormdb
COPY --from=builder /app/build/plugins/ /var/lib/stormdb/plugins/

# Copy configuration files
COPY --from=builder /app/config/core.yaml /app/config/core.yaml.example

# Copy plugin configurations and documentation
COPY --from=builder /app/plugins/*/config-example.yaml /app/config/plugins/
COPY --from=builder /app/plugins/*/API_EXAMPLES.md /app/docs/
COPY --from=builder /app/*.md /app/docs/

# Create default configuration
RUN cp /app/config/core.yaml.example /app/config/core.yaml

# Set proper permissions
RUN chmod +x /usr/local/bin/stormdb && \
    chown -R stormdb:stormdb /app /var/lib/stormdb

# Switch to non-root user
USER stormdb

# Set working directory
WORKDIR /app

# Set environment variables
ENV STORMDB_CONFIG_PATH=/app/config/core.yaml
ENV STORMDB_PLUGIN_DIR=/var/lib/stormdb/plugins
ENV STORMDB_DATA_DIR=/app/data
ENV STORMDB_LOG_DIR=/app/logs

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["stormdb"]
CMD ["-config", "/app/config/core.yaml"]

# Build metadata
LABEL maintainer="StormDB Team <team@stormdb.io>"
LABEL version="0.4-alpha"
LABEL description="StormDB v0.4-alpha - Modular PostgreSQL Performance Testing Platform"
LABEL org.opencontainers.image.title="StormDB v0.4-alpha"
LABEL org.opencontainers.image.description="Modular plugin-based database performance testing tool"
LABEL org.opencontainers.image.version="0.4-alpha"
LABEL org.opencontainers.image.source="https://github.com/elchinoo/stormdb"
LABEL org.opencontainers.image.documentation="https://github.com/elchinoo/stormdb/tree/v2-redesign-core"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.vendor="StormDB Team"
LABEL org.opencontainers.image.authors="StormDB Team"

# Architecture and build info
LABEL org.opencontainers.image.architecture="amd64"
LABEL stormdb.plugins.included="bulk-load,tpcc-scalability"
LABEL stormdb.features="plugin-system,rest-api,postgresql-support,metrics-collection"
