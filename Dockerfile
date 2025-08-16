# ========================================================================
# StormDB Multi-Platform Docker Build Environment
# Supports: Linux (Ubuntu), cross-compilation, development, and debugging
# ========================================================================

# Base development image with all tools (overrideable for China registries)
ARG BASE_IMAGE
FROM ${BASE_IMAGE:-ubuntu:22.04} AS base

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    gcc \
    clang \
    gdb \
    valgrind \
    pkg-config \
    libyaml-dev \
    libpq-dev \
    postgresql-client \
    git \
    make \
    cmake \
    curl \
    wget \
    vim \
    nano \
    htop \
    strace \
    ltrace \
    && rm -rf /var/lib/apt/lists/*

# Set up development environment
WORKDIR /workspace
ENV CC=gcc
ENV CXX=g++

# Development stage - includes debugging tools
FROM base AS development

# Install additional development tools
RUN apt-get update && apt-get install -y \
    gdb-multiarch \
    qemu-user-static \
    libc6-dbg \
    libc6-dev \
    manpages-dev \
    && rm -rf /var/lib/apt/lists/*

# Configure GDB for container debugging
RUN echo "set auto-load safe-path /" >> /root/.gdbinit

# Create development user (optional, for non-root development)
RUN useradd -m -s /bin/bash dev && \
    usermod -aG sudo dev && \
    echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Testing stage - optimized for CI/CD
FROM base AS testing

# Copy source code
COPY . /workspace/

# Build all variants for testing
RUN make clean && \
    make info && \
    make all && \
    make debug && \
    make release && \
    make asan && \
    make ubsan && \
    make tsan

# Run basic tests
RUN make test-asan && \
    make test-ubsan && \
    make test-tsan

# Production stage - minimal runtime (overrideable)
ARG PRODUCTION_BASE_IMAGE
FROM ${PRODUCTION_BASE_IMAGE:-ubuntu:22.04} AS production

# Install only runtime dependencies
RUN apt-get update && apt-get install -y \
    libyaml-0-2 \
    libpq5 \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Create app user for security
RUN useradd -r -s /bin/false stormdb

# Copy built binary
COPY --from=testing /workspace/bin/stormdb-release /usr/local/bin/stormdb

# Set permissions
RUN chmod +x /usr/local/bin/stormdb

# Switch to non-root user
USER stormdb

# Default command
CMD ["stormdb", "--help"]

# Multi-architecture build stage
FROM base AS cross-compile

# Install cross-compilation tools
RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    libc6-dev-arm64-cross \
    libc6-dev-armhf-cross \
    && rm -rf /var/lib/apt/lists/*

# Development with VS Code integration
FROM development AS vscode

# Install VS Code server dependencies
RUN apt-get update && apt-get install -y \
    openssh-server \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH for VS Code remote development
RUN mkdir /var/run/sshd && \
    echo 'root:stormdb' | chpasswd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd

# Expose SSH port
EXPOSE 22

# Start SSH service
CMD ["/usr/sbin/sshd", "-D"]
