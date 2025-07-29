#!/bin/bash

set -e

VERSION="1.0.0"

echo "AX206 System Monitor - Build Script"
echo "==================================="

if ! command -v go &> /dev/null; then
    echo "Error: Go compiler not found, please install Go first"
    exit 1
fi

echo "Go version: $(go version)"

mkdir -p dist

echo "Cleaning previous build files..."
rm -rf dist/*

cd src/ax206monitor

if [ ! -f go.mod ]; then
    echo "Initializing Go module..."
    go mod init ax206monitor
fi

echo "Downloading dependencies..."
go mod tidy

echo "Compiling Linux version with privacy protection..."
GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
    -trimpath \
    -buildmode=exe \
    -o ../../dist/ax206monitor-linux-amd64 .

cd ../..

chmod +x dist/ax206monitor-linux-amd64

echo ""
echo "Compilation complete!"
echo "Output files in dist/ directory:"
ls -la dist/

echo ""
echo "Usage Instructions:"
echo "Linux: ./dist/ax206monitor-linux-amd64"
echo ""
echo "Note: Ensure AX206 device is connected with proper USB permissions before running" 