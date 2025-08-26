#!/bin/bash

echo "=== Go Web Framework Benchmarking Suite - Complete Workflow ==="
echo

# Check for running processes on the configured ports
echo "Checking for processes on framework ports..."
for port in 17780 17781 17782 17783 17784; do
    if lsof -ti:$port > /dev/null 2>&1; then
        echo "Found process on port $port, killing it..."
        lsof -ti:$port | xargs kill -9 || true
    else
        echo "No process found on port $port"
    fi
done
sleep 1

go run cmd/*.go build --clean || exit 1
go run cmd/*.go run
