#!/bin/bash

# Development script with hot reload

# Determine GOPATH
GOPATH=${GOPATH:-$HOME/go}
AIR_BIN="$GOPATH/bin/air"

# Check if air is installed
if [ ! -f "$AIR_BIN" ]; then
    echo "📦 Air not installed. Installing..."
    go install github.com/air-verse/air@latest
fi

echo "🔥 Starting Oubliette in development mode with hot reload..."
echo "   File changes will automatically rebuild and restart the server"
echo ""

"$AIR_BIN" -c .air.toml
