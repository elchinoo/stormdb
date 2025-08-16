#!/bin/bash
# ========================================================================
# StormDB Offline Container Development Session
# Uses existing container images without requiring external network access
# ========================================================================

set -e

# Use your existing UBI image
CONTAINER_IMAGE="b39f0b68f0c6"
PROJECT_DIR="$(pwd)"

echo "🐧 Starting StormDB Linux Development Session"
echo "============================================="

# Start container in background
CONTAINER_ID=$(podman run -d --rm \
    --user root \
    -v "$PROJECT_DIR:/workspace" \
    -w /workspace \
    -e TERM=xterm-256color \
    -e STORMDB_ENV=development \
    "$CONTAINER_IMAGE" \
    sleep infinity)

echo "✅ Container started: $CONTAINER_ID"

# Function to execute commands in container
exec_in_container() {
    podman exec -it "$CONTAINER_ID" bash -c "$1"
}

# Check what's available
echo "🔍 Checking available development tools..."
exec_in_container "
echo 'System Information:'
cat /etc/os-release | head -5
echo ''

echo 'Available Tools:'
which gcc && echo '✅ GCC found' || echo '❌ GCC not found'
which make && echo '✅ Make found' || echo '❌ Make not found'
which gdb && echo '✅ GDB found' || echo '❌ GDB not found'
which git && echo '✅ Git found' || echo '❌ Git not found'

echo ''
echo 'GCC Version:'
gcc --version | head -1 || echo 'GCC not available'

echo ''
echo 'Starting development session...'
echo 'Type \"exit\" to leave the container'
echo ''
"

# Start interactive session
echo "🚀 Entering Linux development environment..."
exec_in_container "cd /workspace && exec bash"

# Cleanup (this runs when container exits)
echo "🧹 Cleaning up container..."
podman stop "$CONTAINER_ID" 2>/dev/null || true
