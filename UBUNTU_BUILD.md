# Ubuntu 24 Build Instructions

## Prerequisites

### 1. Install Go 1.23+ (Required for latest tools)
```bash
# Remove old Go version if installed via apt
sudo apt remove golang-go

# Install Go 1.23+ manually
wget https://go.dev/dl/go1.23.11.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.11.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.profile)
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Reload environment
source ~/.bashrc
```

### 2. Install system dependencies
```bash
sudo apt update
sudo apt install -y build-essential git curl wget
```

### 3. Verify Go installation
```bash
go version  # Should show go1.23.11 or later
```

## Build StormDB

### Option 1: Skip dev-tools (Recommended for Ubuntu)
```bash
# Clone the repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Build without dev-tools
make deps
make build
make plugins

# Verify build
./build/stormdb --help
```

### Option 2: Install dev-tools manually
```bash
# If you need development tools, install them manually:
go install golang.org/x/tools/cmd/godoc@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# For gosec, use a specific working version:
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@v2.21.4

go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/air-verse/air@latest
```

### Option 3: Use alternative security scanner
```bash
# Instead of gosec, you can use staticcheck:
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## Troubleshooting

### Git Authentication Error
If you get "could not read Username for 'https://github.com'" error:

```bash
# Configure Git to use HTTPS instead of SSH
git config --global url."https://github.com/".insteadOf git@github.com:

# Or set credential helper
git config --global credential.helper store
```

### Network/Proxy Issues
If behind a corporate firewall:

```bash
# Set Go proxy
go env -w GOPROXY=https://proxy.golang.org,direct
go env -w GOSUMDB=sum.golang.org

# Or use Athens proxy
go env -w GOPROXY=https://athens.azurefd.net
```

### Module Download Issues
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download
```

## Quick Start Commands

```bash
# Complete build process
make deps
make build
make plugins

# Test the build
./build/stormdb -c config/config_simple_connection.yaml --help

# Run a quick test (adjust connection details)
./build/stormdb -c config/config_simple_connection.yaml -duration=10s
```
