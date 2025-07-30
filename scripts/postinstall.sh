#!/bin/bash
# Post-installation script for StormDB

set -e

echo "Installing StormDB..."

# Set proper permissions on binary
chmod +x /usr/bin/stormdb

# Set permissions on plugins
if [ -d /usr/lib/stormdb/plugins ]; then
    chmod -R 755 /usr/lib/stormdb/plugins
    echo "Plugins installed to /usr/lib/stormdb/plugins/"
fi

# Set permissions on configuration files
if [ -d /etc/stormdb ]; then
    chmod -R 644 /etc/stormdb/*.yaml
    chmod 755 /etc/stormdb
    echo "Configuration files installed to /etc/stormdb/"
fi

# Set permissions on example configurations
if [ -d /etc/stormdb/examples ]; then
    chmod -R 644 /etc/stormdb/examples/*.yaml
    chmod 755 /etc/stormdb/examples
    echo "Example configurations available in /etc/stormdb/examples/"
fi

# Update man database
if command -v mandb >/dev/null 2>&1; then
    mandb -q 2>/dev/null || true
elif command -v makewhatis >/dev/null 2>&1; then
    makewhatis /usr/share/man 2>/dev/null || true
fi

# Display installation summary
echo ""
echo "StormDB installation completed successfully!"
echo ""
echo "Usage:"
echo "  stormdb --help                                   # Show help"
echo "  stormdb -c /etc/stormdb/config_tpcc.yaml        # Run TPC-C workload"
echo "  stormdb --workload ecommerce_mixed --duration 30s # Run e-commerce workload"
echo ""
echo "Files installed:"
echo "  Binary:        /usr/bin/stormdb"
echo "  Configuration: /etc/stormdb/config_tpcc.yaml"
echo "  Examples:      /etc/stormdb/examples/"
echo "  Plugins:       /usr/lib/stormdb/plugins/"
echo "  Documentation: /usr/share/doc/stormdb/"
echo "  Manual:        man stormdb"
echo ""
echo "Documentation: https://github.com/elchinoo/stormdb"
