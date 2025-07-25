#!/bin/bash

# Test script for mv2s3

set -e

echo "Running mv2s3 test suite..."
echo "================================"

# Change to project root
cd "$(dirname "$0")/.."

# Run go mod tidy to ensure dependencies are up to date
echo "Updating dependencies..."
go mod tidy

# Run tests with coverage
echo "Running unit tests with coverage..."
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Display coverage report
echo "Coverage report:"
go tool cover -func=coverage.out

# Generate HTML coverage report
echo "Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report saved to coverage.html"

# Run specific test categories
echo ""
echo "Running tests by category..."

echo "- Core types tests:"
go test -v ./pkg/types/

echo "- Configuration tests:"
go test -v ./internal/config/

echo "- CLI tests:"
go test -v ./internal/cli/

echo "- Scanner tests:"
go test -v ./internal/scanner/

echo "- Storage tests:"
go test -v ./internal/storage/

echo "- Processor tests:"
go test -v ./internal/processor/

echo "- Integration tests:"
go test -v ./test/integration/

echo ""
echo "All tests completed successfully!"
