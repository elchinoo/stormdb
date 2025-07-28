#!/bin/bash
# Post-removal script for StormDB

set -e

# Stop and disable service if systemd is available
if command -v systemctl >/dev/null 2>&1; then
    systemctl stop stormdb.service || true
    systemctl disable stormdb.service || true
    systemctl daemon-reload || true
fi

# Remove symlinks
rm -f /var/lib/stormdb/plugins/system

# Ask user if they want to remove data (interactive removal)
if [[ -t 0 ]]; then  # Check if running interactively
    echo "Do you want to remove StormDB data and logs? (y/N)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        rm -rf /var/lib/stormdb
        rm -rf /var/log/stormdb
        echo "StormDB data and logs removed."
    else
        echo "StormDB data and logs preserved in /var/lib/stormdb and /var/log/stormdb"
    fi
else
    # Non-interactive removal - preserve data
    echo "StormDB data and logs preserved in /var/lib/stormdb and /var/log/stormdb"
    echo "Remove manually if no longer needed."
fi

# Note: We don't remove the stormdb user as it might be used by other processes
echo "StormDB package removed successfully!"
echo "Note: The 'stormdb' system user was preserved for safety."
