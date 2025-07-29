#!/bin/bash
# Fix script for StormDB DEB installation issues
# Run this on the Ubuntu system where StormDB was installed

set -e

echo "🔧 StormDB Installation Fix Script"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    print_error "This script must be run as root (use sudo)"
    exit 1
fi

print_status "Checking current StormDB installation..."

# Fix directory structure
print_status "Fixing directory structure..."

# Remove the incorrectly created directory
if [ -d "/var/lib/stormdb/{config,logs,plugins}" ]; then
    print_warning "Removing incorrectly created directory: /var/lib/stormdb/{config,logs,plugins}"
    rm -rf "/var/lib/stormdb/{config,logs,plugins}"
fi

# Ensure correct directories exist
mkdir -p /var/lib/stormdb/{config,logs,plugins}
mkdir -p /var/log/stormdb

# Set proper ownership
chown -R stormdb:stormdb /var/lib/stormdb
chown -R stormdb:stormdb /var/log/stormdb

print_status "Directory structure fixed"

# Create default configuration if missing
if [ ! -f "/var/lib/stormdb/config/stormdb.yaml" ]; then
    print_status "Creating default configuration file..."
    
    cat > /var/lib/stormdb/config/stormdb.yaml << 'EOF'
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: ""
  dbname: "postgres"
  sslmode: "disable"

workload:
  type: "basic"
  duration: "30s"
  workers: 4

metrics:
  enabled: true
  interval: "5s"
EOF
    
    chown stormdb:stormdb /var/lib/stormdb/config/stormdb.yaml
    chmod 640 /var/lib/stormdb/config/stormdb.yaml
    print_status "Default configuration created"
else
    print_status "Configuration file already exists"
fi

# Check if binary exists and is executable
if [ -f "/usr/local/bin/stormdb" ]; then
    chmod +x /usr/local/bin/stormdb
    print_status "Binary permissions fixed"
else
    print_error "StormDB binary not found at /usr/local/bin/stormdb"
fi

# Fix systemd service if it exists
if [ -f "/etc/systemd/system/stormdb.service" ]; then
    print_status "Reloading systemd daemon..."
    systemctl daemon-reload
    systemctl enable stormdb.service || true
    print_status "SystemD service configuration updated"
fi

# Test the installation
print_status "Testing StormDB installation..."

if command -v stormdb >/dev/null 2>&1; then
    print_status "✅ StormDB binary is accessible"
    
    # Test help command
    if stormdb --help >/dev/null 2>&1; then
        print_status "✅ StormDB help command works"
    else
        print_warning "⚠️  StormDB help command failed"
    fi
else
    print_error "❌ StormDB binary not found in PATH"
    echo "You may need to add /usr/local/bin to your PATH"
fi

# Show final status
echo ""
print_status "🎉 Installation fix complete!"
echo ""
echo "Summary:"
echo "├── Configuration: /var/lib/stormdb/config/stormdb.yaml"
echo "├── Logs directory: /var/lib/stormdb/logs/"
echo "├── Binary: /usr/local/bin/stormdb"
echo "└── Service: systemctl {start|stop|status} stormdb"
echo ""
echo "Next steps:"
echo "1. Edit the configuration file with your database details:"
echo "   sudo -u stormdb nano /var/lib/stormdb/config/stormdb.yaml"
echo ""
echo "2. Test the configuration:"
echo "   stormdb -c /var/lib/stormdb/config/stormdb.yaml --help"
echo ""
echo "3. Run a quick test (if database is configured):"
echo "   stormdb -c /var/lib/stormdb/config/stormdb.yaml -duration=10s"
