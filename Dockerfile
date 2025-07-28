# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies including C compiler for CGO
RUN apk add --no-cache git ca-certificates tzdata make gcc musl-dev binutils-gold

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build plugins first (requires CGO)
RUN CGO_ENABLED=1 make plugins

# Build the application
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata postgresql-client

# Create app user
RUN addgroup -g 1000 -S stormdb && \
    adduser -u 1000 -S stormdb -G stormdb

# Create necessary directories
RUN mkdir -p /app/config /app/plugins /app/data && \
    chown -R stormdb:stormdb /app

# Copy binary and plugins from builder
COPY --from=builder /app/build/stormdb /usr/local/bin/stormdb
COPY --from=builder /app/build/plugins/*.so /app/plugins/
COPY --from=builder /app/config/ /app/config/

# Copy documentation
COPY --from=builder /app/README.md /app/ARCHITECTURE.md /app/docs/ /app/docs/

# Set proper permissions
RUN chmod +x /usr/local/bin/stormdb && \
    chown -R stormdb:stormdb /app

# Switch to non-root user
USER stormdb

# Set working directory
WORKDIR /app

# Expose ports (if needed for monitoring/metrics)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD stormdb --help || exit 1

# Default command
ENTRYPOINT ["stormdb"]
CMD ["--help"]

# Labels
LABEL maintainer="StormDB Team"
LABEL version="1.0.0"
LABEL description="PostgreSQL performance testing and benchmarking tool"
LABEL org.opencontainers.image.source="https://github.com/elchinoo/stormdb"
LABEL org.opencontainers.image.documentation="https://github.com/elchinoo/stormdb/docs"
LABEL org.opencontainers.image.licenses="MIT"
