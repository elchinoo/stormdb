#!/bin/bash

# Test script for StormDB v2 Web Server
set -e

echo "Starting StormDB v2 Web Server Integration Test..."

# Build the application
echo "Building StormDB..."
cd /Users/charly.batista/proj/pgstorm/stormdb/v2
go build -o ./build/stormdb ./cmd/stormdb

# Start the server in background
echo "Starting server..."
./build/stormdb &
SERVER_PID=$!

# Give server time to start
sleep 2

# Test health endpoint
echo "Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
echo "Health response: $HEALTH_RESPONSE"

# Test status endpoint
echo "Testing status endpoint..."
STATUS_RESPONSE=$(curl -s http://localhost:8080/status)
echo "Status response: $STATUS_RESPONSE"

# Test plugins endpoint
echo "Testing plugins endpoint..."
PLUGINS_RESPONSE=$(curl -s http://localhost:8080/plugins)
echo "Plugins response: $PLUGINS_RESPONSE"

# Test scheduler status endpoint
echo "Testing scheduler status endpoint..."
SCHEDULER_RESPONSE=$(curl -s http://localhost:8080/scheduler/status)
echo "Scheduler response: $SCHEDULER_RESPONSE"

# Test invalid endpoint (should return 404)
echo "Testing invalid endpoint..."
INVALID_RESPONSE=$(curl -s -w "%{http_code}" http://localhost:8080/invalid)
echo "Invalid endpoint response code: $INVALID_RESPONSE"

# Cleanup
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo "Integration test completed successfully!"
