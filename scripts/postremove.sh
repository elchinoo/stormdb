#!/bin/bash
# Post-removal script for StormDB

set -e

echo "Removing StormDB..."

# Update man database
if command -v mandb >/dev/null 2>&1; then
    mandb -q 2>/dev/null || true
elif command -v makewhatis >/dev/null 2>&1; then
    makewhatis /usr/share/man 2>/dev/null || true
fi

# Ask user if they want to remove configuration files (interactive removal)
if [[ -t 0 ]]; then  # Check if running interactively
    echo "Do you want to remove StormDB configuration files from /etc/stormdb? (y/N)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        rm -rf /etc/stormdb
        echo "StormDB configuration files removed."
    else
        echo "StormDB configuration files preserved in /etc/stormdb"
    fi
else
    # Non-interactive removal - preserve configuration
    echo "StormDB configuration files preserved in /etc/stormdb"
    echo "Remove manually if no longer needed: rm -rf /etc/stormdb"
fi

echo "StormDB package removed successfully!"
