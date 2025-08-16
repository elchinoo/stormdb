#!/bin/bash
# ========================================================================
# StormDB Quick Linux Development Session
# Uses your existing stormdb-builder-ubuntu image with proper setup
# ========================================================================

set -e

PROJECT_DIR="$(pwd)"
IMAGE_NAME="stormdb-builder-ubuntu:latest"

echo "🐧 Starting StormDB Linux Development Session"
echo "============================================="

# Start interactive container with all dependencies
docker run -it --rm \
    --platform linux/amd64 \
    -v "$PROJECT_DIR:/workspace" \
    -w /workspace \
    -e TERM=xterm-256color \
    -e STORMDB_ENV=development \
    "$IMAGE_NAME" \
    bash -c "
# Install missing dependencies
echo '🔧 Installing development dependencies...'
apt-get update -qq && apt-get install -y -qq \
    libyaml-dev \
    libpq-dev \
    pkg-config \
    gdb \
    valgrind

echo '🔍 Development environment ready!'
echo 'Available tools:'
which gcc && echo '✅ GCC found'
which make && echo '✅ Make found'  
which gdb && echo '✅ GDB found'
which valgrind && echo '✅ Valgrind found'

echo ''
echo 'System info:'
uname -a
echo ''

echo '💡 Quick commands:'
echo '  make all       - Build all variants'
echo '  make debug     - Build debug version'
echo '  make asan      - Build with AddressSanitizer' 
echo '  gdb bin/stormdb-debug  - Debug with GDB'
echo ''

echo '🚀 Starting Linux development shell...'
echo 'Type \"exit\" to return to macOS'
echo ''

# Start interactive bash session
exec bash
"
