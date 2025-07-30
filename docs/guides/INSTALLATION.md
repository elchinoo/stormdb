# Installation Guide

This guide provides detailed installation instructions for StormDB on various platforms.

## System Requirements

### Minimum Requirements
- **Operating System**: Linux, macOS, or Windows
- **Architecture**: x86_64 (amd64) or ARM64
- **Memory**: 1GB RAM minimum, 4GB recommended
- **Disk Space**: 100MB for binary and plugins
- **PostgreSQL**: Version 12 or higher

### Build Requirements (Source Installation)
- **Go**: Version 1.24 or higher
- **Git**: For cloning the repository
- **Make**: Build automation
- **GCC/Clang**: For plugin compilation (Linux/macOS)
- **MSVC**: For plugin compilation (Windows)

### Database Requirements
- PostgreSQL 12+ with superuser access for initial setup
- Network connectivity to PostgreSQL instance
- Sufficient disk space for test data (varies by workload)

## Installation Methods

### Method 1: Pre-built Binaries (Recommended)

The easiest way to get started is with pre-built binaries:

```bash
# Linux x86_64
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-amd64 -o stormdb
chmod +x stormdb

# Linux ARM64
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-arm64 -o stormdb
chmod +x stormdb

# macOS x86_64 (Intel)
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-darwin-amd64 -o stormdb
chmod +x stormdb

# macOS ARM64 (Apple Silicon)
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-darwin-arm64 -o stormdb
chmod +x stormdb

# Windows
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-windows-amd64.exe -o stormdb.exe
```

#### Installing to System PATH

```bash
# Linux/macOS
sudo mv stormdb /usr/local/bin/
sudo chmod +x /usr/local/bin/stormdb

# Verify installation
stormdb --version
```

### Method 2: Build from Source

#### Prerequisites Installation

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install -y git make gcc build-essential
```

**CentOS/RHEL/Fedora:**
```bash
sudo yum install -y git make gcc
# OR for newer versions:
sudo dnf install -y git make gcc
```

**macOS:**
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install Go using Homebrew (optional)
brew install go git make
```

**Windows:**
- Install Git for Windows
- Install Go from https://golang.org/dl/
- Install Make (via chocolatey: `choco install make`)
- Install Build Tools for Visual Studio

#### Clone and Build

```bash
# Clone the repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Install Go dependencies
go mod download

# Build the binary
make build

# Build all plugins
make build-plugins

# Or build everything at once
make build-all

# Install to system PATH (optional)
sudo make install
```

#### Development Build with Tools

```bash
# Install development tools (linters, formatters, etc.)
make dev-tools

# Run tests
make test

# Run with coverage
make test-coverage

# Build with debug symbols
make build-debug
```

### Method 3: Docker

#### Using Pre-built Images

```bash
# Pull the latest image
docker pull elchinoo/stormdb:latest

# Run with a configuration file
docker run --rm \
  -v $(pwd)/config:/config \
  -v $(pwd)/results:/results \
  elchinoo/stormdb:latest \
  --config /config/my-config.yaml
```

#### Building Custom Docker Image

```bash
# Clone repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Build Docker image
docker build -t stormdb:custom .

# Build with specific Go version
docker build --build-arg GO_VERSION=1.24 -t stormdb:custom .
```

#### Docker Compose for Development

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: stormdb_test
      POSTGRES_USER: stormdb_user
      POSTGRES_PASSWORD: stormdb_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  stormdb:
    build: .
    depends_on:
      - postgres
    volumes:
      - ./config:/config
      - ./results:/results
    command: ["--config", "/config/docker-config.yaml"]

volumes:
  postgres_data:
```

### Method 4: Package Managers

#### Homebrew (macOS/Linux)

```bash
# Add tap (when available)
brew tap elchinoo/stormdb
brew install stormdb
```

#### Arch Linux (AUR)

```bash
# Using yay
yay -S stormdb

# Using makepkg
git clone https://aur.archlinux.org/stormdb.git
cd stormdb
makepkg -si
```

## Plugin Installation

### Automatic Plugin Building

```bash
# Build all available plugins
make build-plugins

# Build specific plugin
make build-plugin PLUGIN=imdb_plugin

# Install plugins to default location
make install-plugins
```

### Manual Plugin Installation

```bash
# Download pre-built plugins
mkdir -p plugins
cd plugins

# Download IMDB plugin
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/imdb_plugin.so -o imdb_plugin.so

# Download Vector plugin
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/vector_plugin.so -o vector_plugin.so

# Download E-commerce plugin
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/ecommerce_plugin.so -o ecommerce_plugin.so
```

### Plugin Directory Setup

StormDB searches for plugins in these locations:
1. `./plugins/` (current directory)
2. `~/.stormdb/plugins/` (user directory)
3. `/usr/local/lib/stormdb/plugins/` (system directory)
4. Directory specified by `STORMDB_PLUGIN_PATH` environment variable

```bash
# Create user plugin directory
mkdir -p ~/.stormdb/plugins

# Set custom plugin path
export STORMDB_PLUGIN_PATH="/path/to/my/plugins"
```

## Verification

### Test Installation

```bash
# Check version
stormdb --version

# List available workloads (should show built-in types)
stormdb --list-workloads

# Validate configuration
stormdb --config config.yaml --validate

# Test database connection
stormdb --config config.yaml --test-connection
```

### Verify Plugin Loading

```bash
# List all available workloads (including plugins)
stormdb --list-workloads

# Check plugin information
stormdb --plugin-info

# Scan for plugins in directories
stormdb --scan-plugins
```

## PostgreSQL Setup

### Database Creation

```sql
-- Connect as superuser (postgres)
psql -U postgres

-- Create database and user
CREATE DATABASE stormdb_test;
CREATE USER stormdb_user WITH PASSWORD 'your_secure_password';

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE stormdb_test TO stormdb_user;
GRANT CREATE ON SCHEMA public TO stormdb_user;

-- For monitoring features (optional)
GRANT SELECT ON ALL TABLES IN SCHEMA pg_catalog TO stormdb_user;
GRANT SELECT ON ALL TABLES IN SCHEMA information_schema TO stormdb_user;

-- For pg_stat_statements (optional)
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

### PostgreSQL Configuration

Add to `postgresql.conf` for optimal testing:

```ini
# Performance settings for testing
shared_preload_libraries = 'pg_stat_statements'
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB

# Monitoring settings
track_activities = on
track_counts = on
track_io_timing = on
track_functions = all
track_activity_query_size = 4096

# pg_stat_statements settings
pg_stat_statements.max = 10000
pg_stat_statements.track = all
pg_stat_statements.save = on
```

## Troubleshooting

### Common Installation Issues

**Permission Denied:**
```bash
# Fix executable permissions
chmod +x stormdb

# Fix plugin permissions
chmod +x plugins/*.so
```

**Plugin Loading Errors:**
```bash
# Check plugin architecture matches binary
file stormdb
file plugins/*.so

# Verify plugin dependencies
ldd plugins/imdb_plugin.so  # Linux
otool -L plugins/imdb_plugin.so  # macOS
```

**Build Errors:**
```bash
# Clean build cache
make clean
go clean -modcache

# Update dependencies
go mod tidy
go mod download

# Rebuild
make build-all
```

**Database Connection Issues:**
```bash
# Test PostgreSQL connection
psql -h localhost -U stormdb_user -d stormdb_test

# Check PostgreSQL logs
tail -f /var/log/postgresql/postgresql-*.log
```

### Getting Help

- Check the [Troubleshooting Guide](TROUBLESHOOTING.md) for specific issues
- Review [GitHub Issues](https://github.com/elchinoo/stormdb/issues) for known problems
- Join [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions) for community support

## Next Steps

After installation, see:
- [Configuration Guide](CONFIGURATION.md) - Set up your first tests
- [Usage Guide](USAGE.md) - Learn command-line options
- [Plugin System Guide](PLUGIN_SYSTEM.md) - Work with plugins
- [Examples](../examples/) - Sample configurations and use cases
