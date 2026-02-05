#!/bin/bash
# Load Test Runner for Oubliette
# Usage: ./run_load_test.sh [duration] [concurrent_users]

set -e

DURATION="${1:-30s}"
CONCURRENT="${2:-10}"
SERVER_URL="${OUBLIETTE_SERVER_URL:-http://localhost:8080}"

echo "=== Oubliette Load Test ==="
echo "Server:     $SERVER_URL"
echo "Duration:   $DURATION"
echo "Concurrent: $CONCURRENT users"
echo ""

# Check if server is running
if ! curl -s "$SERVER_URL/health" > /dev/null 2>&1; then
    echo "ERROR: Server not responding at $SERVER_URL"
    echo "Start the server first: cd ../.. && go run ./cmd/server"
    exit 1
fi

echo "Server is healthy. Starting load test..."
echo ""

# Export config for tests
export OUBLIETTE_SERVER_URL="$SERVER_URL"
export OUBLIETTE_LOAD_DURATION="$DURATION"
export OUBLIETTE_LOAD_CONCURRENT="$CONCURRENT"

# Run tests with profiling
cd "$(dirname "$0")/../.."

echo "--- Health Endpoint Load Test ---"
go test -v -run TestHealthEndpoint ./test/load/... -timeout 45m

if [ -n "$OUBLIETTE_AUTH_TOKEN" ]; then
    echo ""
    echo "--- Project List Load Test ---"
    go test -v -run TestProjectList ./test/load/... -timeout 45m
    
    if [ -n "$OUBLIETTE_TEST_PROJECT" ]; then
        echo ""
        echo "--- Concurrent Sessions Load Test ---"
        go test -v -run TestConcurrentSessions ./test/load/... -timeout 45m
    else
        echo "Skipping session test (OUBLIETTE_TEST_PROJECT not set)"
    fi
else
    echo "Skipping authenticated tests (OUBLIETTE_AUTH_TOKEN not set)"
fi

echo ""
echo "--- Benchmarks ---"
go test -bench=. -benchtime=10s ./test/load/... 2>/dev/null || true

echo ""
echo "=== Load Test Complete ==="
