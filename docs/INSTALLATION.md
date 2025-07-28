# StormDB Installation Guide

This guide covers all available installation methods for StormDB.

## Table of Contents

- [Quick Install (Recommended)](#quick-install-recommended)
- [Download Pre-built Binaries](#download-pre-built-binaries)
- [Linux Package Installation](#linux-package-installation)
- [Docker Installation](#docker-installation)
- [Build from Source](#build-from-source)
- [Verification](#verification)

## Quick Install (Recommended)

### Linux/macOS (using curl)

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/elchinoo/stormdb/main/install.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/elchinoo/stormdb/main/install.sh | bash -s v0.1.0-alpha.2
```

### Windows (using PowerShell)

```powershell
# Install latest version
iwr https://raw.githubusercontent.com/elchinoo/stormdb/main/install.ps1 | iex

# Install specific version  
iwr https://raw.githubusercontent.com/elchinoo/stormdb/main/install.ps1 | iex -Args "v0.1.0-alpha.2"
```

## Download Pre-built Binaries

### GitHub Releases Page

Visit: **https://github.com/elchinoo/stormdb/releases**

#### Linux
```bash
# AMD64 (most common)
wget https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-linux-amd64.tar.gz
tar -xzf stormdb-linux-amd64.tar.gz
sudo mv stormdb /usr/local/bin/

# ARM64 (Raspberry Pi, ARM servers)
wget https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-linux-arm64.tar.gz
tar -xzf stormdb-linux-arm64.tar.gz
sudo mv stormdb /usr/local/bin/

# 32-bit systems
wget https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-linux-386.tar.gz
tar -xzf stormdb-linux-386.tar.gz
sudo mv stormdb /usr/local/bin/
```

#### macOS
```bash
# Intel Macs
curl -L https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-darwin-amd64.tar.gz | tar -xz
sudo mv stormdb /usr/local/bin/

# Apple Silicon Macs (M1/M2/M3)
curl -L https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-darwin-arm64.tar.gz | tar -xz
sudo mv stormdb /usr/local/bin/
```

#### Windows
```powershell
# Download and extract
Invoke-WebRequest -Uri "https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-windows-amd64.zip" -OutFile "stormdb.zip"
Expand-Archive -Path "stormdb.zip" -DestinationPath "C:\Program Files\StormDB"

# Add to PATH (run as Administrator)
$env:PATH += ";C:\Program Files\StormDB"
```

## Linux Package Installation

### Ubuntu/Debian (.deb packages)

```bash
# Download DEB package
wget https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb_0.1.0-alpha.2_amd64.deb

# Install with dpkg
sudo dpkg -i stormdb_0.1.0-alpha.2_amd64.deb

# Fix dependencies if needed
sudo apt-get install -f

# Or install directly
curl -L https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb_0.1.0-alpha.2_amd64.deb -o stormdb.deb && sudo dpkg -i stormdb.deb
```

### CentOS/RHEL/Fedora (.rpm packages)

```bash
# Download RPM package
wget https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-0.1.0-alpha.2.x86_64.rpm

# Install with rpm
sudo rpm -i stormdb-0.1.0-alpha.2.x86_64.rpm

# Or install with yum/dnf
sudo yum install https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-0.1.0-alpha.2.x86_64.rpm

# Fedora with dnf
sudo dnf install https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-0.1.0-alpha.2.x86_64.rpm
```

### Amazon Linux 2

```bash
# Install RPM package
sudo yum install https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-0.1.0-alpha.2.x86_64.rpm
```

## Docker Installation

### GitHub Container Registry (Recommended)

```bash
# Pull latest version
docker pull ghcr.io/elchinoo/stormdb:latest

# Pull specific version
docker pull ghcr.io/elchinoo/stormdb:v0.1.0-alpha.2

# Run with configuration
docker run --rm -v $(pwd)/config:/config ghcr.io/elchinoo/stormdb:latest -c /config/my-workload.yaml

# Interactive mode
docker run --rm -it ghcr.io/elchinoo/stormdb:latest --help
```

### Docker Hub (Alternative)

```bash
# Pull from Docker Hub
docker pull elchinoo/stormdb:latest
docker pull elchinoo/stormdb:v0.1.0-alpha.2
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  stormdb:
    image: ghcr.io/elchinoo/stormdb:latest
    volumes:
      - ./config:/config
    command: ["-c", "/config/config_ecommerce_mixed.yaml", "-d", "60s"]
    depends_on:
      - postgres
  
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: storm
      POSTGRES_USER: stormdb
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
```

## Build from Source

### Prerequisites

- Go 1.21+
- Git
- Make

### Clone and Build

```bash
# Clone repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Install dependencies
make deps

# Build binary and plugins
make build-all

# Run tests
make test

# Install to system
make install
```

### Development Build

```bash
# Install development tools
make dev-tools

# Build development version with debug info
make build-dev

# Watch for changes and rebuild
make dev-watch
```

## Verification

### Verify Installation

```bash
# Check version
stormdb --version

# Show help
stormdb --help

# List available workloads
stormdb --list-workloads
```

### Verify Package Integrity

```bash
# Download checksums
wget https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/SHA256SUMS

# Verify binary
sha256sum -c SHA256SUMS --ignore-missing
```

### Verify Docker Image

```bash
# Check image details
docker inspect ghcr.io/elchinoo/stormdb:latest

# Verify signature (if available)
docker trust inspect ghcr.io/elchinoo/stormdb:latest
```

## Package Details

### Linux Packages Include

- **Binary**: `/usr/local/bin/stormdb`
- **Config files**: `/etc/stormdb/`
- **Documentation**: `/usr/share/doc/stormdb/`
- **Systemd service**: `stormdb.service`
- **User management**: Automatic `stormdb` user creation

### Service Management (Linux packages)

```bash
# Start service
sudo systemctl start stormdb

# Enable on boot
sudo systemctl enable stormdb

# Check status
sudo systemctl status stormdb

# View logs
sudo journalctl -u stormdb -f
```

## Platform Support

| Platform | Architecture | Binary | Package | Docker |
|----------|-------------|--------|---------|---------|
| Linux | amd64 | ✅ | DEB/RPM | ✅ |
| Linux | arm64 | ✅ | DEB/RPM | ✅ |
| Linux | 386 | ✅ | DEB/RPM | ❌ |
| macOS | amd64 (Intel) | ✅ | ❌ | ❌ |
| macOS | arm64 (M1/M2/M3) | ✅ | ❌ | ❌ |
| Windows | amd64 | ✅ | ❌ | ❌ |
| Windows | 386 | ✅ | ❌ | ❌ |

## Troubleshooting

### Common Issues

#### Permission Denied (Linux/macOS)
```bash
# Make binary executable
chmod +x stormdb

# Move to system path
sudo mv stormdb /usr/local/bin/
```

#### Windows Execution Policy
```powershell
# Allow script execution
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### Package Installation Fails
```bash
# Ubuntu/Debian: Fix dependencies
sudo apt-get update && sudo apt-get install -f

# CentOS/RHEL: Clear cache
sudo yum clean all && sudo yum makecache
```

#### Docker Permission Issues
```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Logout and login again
```

### Getting Help

- **Documentation**: https://github.com/elchinoo/stormdb/blob/main/README.md
- **Issues**: https://github.com/elchinoo/stormdb/issues
- **Discussions**: https://github.com/elchinoo/stormdb/discussions
- **Security**: https://github.com/elchinoo/stormdb/security/advisories

## Update/Upgrade

### Binary Updates
```bash
# Download new version and replace
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-amd64.tar.gz | tar -xz
sudo mv stormdb /usr/local/bin/
```

### Package Updates
```bash
# Ubuntu/Debian
sudo dpkg -i stormdb_<new-version>_amd64.deb

# CentOS/RHEL/Fedora
sudo rpm -U stormdb-<new-version>.x86_64.rpm
```

### Docker Updates
```bash
# Pull latest image
docker pull ghcr.io/elchinoo/stormdb:latest

# Or specific version
docker pull ghcr.io/elchinoo/stormdb:v0.1.0-alpha.3
```

---

**Need help?** Open an issue at https://github.com/elchinoo/stormdb/issues
