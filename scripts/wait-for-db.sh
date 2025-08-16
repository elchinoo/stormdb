#!/usr/bin/env bash
# Wait for TCP port to be available
set -euo pipefail
HOST=${1:-127.0.0.1}
PORT=${2:-54320}
TIMEOUT=${3:-30}

SECONDS=0
while ! nc -z "$HOST" "$PORT"; do
  if [ "$SECONDS" -ge "$TIMEOUT" ]; then
    echo "Timed out waiting for $HOST:$PORT" >&2
    exit 1
  fi
  sleep 1
done

echo "$HOST:$PORT reachable"
