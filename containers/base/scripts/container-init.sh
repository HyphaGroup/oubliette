#!/bin/bash
# Container initialization script for Oubliette containers
# Starts the relay in the background, then exec's the main command

set -e

# Start the relay in the background if project ID is set
if [ -n "$OUBLIETTE_PROJECT_ID" ]; then
    echo "Starting oubliette-relay for project $OUBLIETTE_PROJECT_ID..."
    /usr/local/bin/oubliette-relay &
    RELAY_PID=$!
    echo "Relay started with PID $RELAY_PID"
fi

# Execute the main command (passed as arguments)
exec "$@"
