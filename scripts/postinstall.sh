#!/bin/bash
# Post-installation script for StormDB

set -e

# Create stormdb user if it doesn't exist
if ! id "stormdb" &>/dev/null; then
    useradd --system --home-dir /var/lib/stormdb --create-home --shell /bin/false --user-group stormdb
fi

# Create necessary directories
mkdir -p /var/lib/stormdb/{config,logs,plugins}
mkdir -p /var/log/stormdb

# Set proper permissions
chown -R stormdb:stormdb /var/lib/stormdb
chown -R stormdb:stormdb /var/log/stormdb
chmod 755 /var/lib/stormdb
chmod 755 /var/log/stormdb

# Set permissions on binary
chmod +x /usr/local/bin/stormdb

# Set permissions on plugins
if [[ -d /usr/local/lib/stormdb/plugins ]]; then
    chmod -R 755 /usr/local/lib/stormdb/plugins
fi

# Create symlink for plugins in user directory
if [[ -d /usr/local/lib/stormdb/plugins ]]; then
    ln -sf /usr/local/lib/stormdb/plugins /var/lib/stormdb/plugins/system
fi

# Copy default config if it doesn't exist
if [[ ! -f /var/lib/stormdb/config/stormdb.yaml ]] && [[ -d /etc/stormdb ]]; then
    cp /etc/stormdb/config_simple_connection.yaml /var/lib/stormdb/config/stormdb.yaml
    chown stormdb:stormdb /var/lib/stormdb/config/stormdb.yaml
fi

# Enable systemd service if systemd is available
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable stormdb.service || true
    echo "StormDB service enabled. Use 'systemctl start stormdb' to start."
fi

echo "StormDB installation completed successfully!"
echo ""
echo "Quick start:"
echo "  1. Edit configuration: /var/lib/stormdb/config/stormdb.yaml"
echo "  2. Run benchmark: stormdb -c /var/lib/stormdb/config/stormdb.yaml"
echo "  3. View help: stormdb --help"
echo ""
echo "Documentation: https://github.com/elchinoo/stormdb"
