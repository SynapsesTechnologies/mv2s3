#!/bin/bash

# Build script for mv2s3

set -e

echo "Building mv2s3..."
echo "===================="

# Change to project root
cd "$(dirname "$0")/.."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build for current platform
echo "Building for current platform..."
go build -o bin/mv2s3 ./cmd/mv2s3

echo "Build completed: bin/mv2s3"

# Optionally build for multiple platforms
if [[ "$1" == "all" ]]; then
    echo "Building for multiple platforms..."
    
    # Linux
    echo "Building for Linux (amd64)..."
    GOOS=linux GOARCH=amd64 go build -o bin/mv2s3-linux-amd64 ./cmd/mv2s3
    
    # macOS
    echo "Building for macOS (amd64)..."
    GOOS=darwin GOARCH=amd64 go build -o bin/mv2s3-darwin-amd64 ./cmd/mv2s3
    
    # macOS ARM64
    echo "Building for macOS (arm64)..."
    GOOS=darwin GOARCH=arm64 go build -o bin/mv2s3-darwin-arm64 ./cmd/mv2s3
    
    # Windows
    echo "Building for Windows (amd64)..."
    GOOS=windows GOARCH=amd64 go build -o bin/mv2s3-windows-amd64.exe ./cmd/mv2s3
    
    echo "Multi-platform builds completed!"
    ls -la bin/
fi
