#!/bin/bash

set -e

VERSION="1.0.0"
PACKAGE_NAME="ax206monitor-linux-amd64-v${VERSION}"

echo "AX206 System Monitor - Package Script"
echo "====================================="
echo "Version: $VERSION"
echo "Package: $PACKAGE_NAME"
echo ""

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

echo "Creating package directory..."
rm -rf "$PACKAGE_NAME"
mkdir -p "$PACKAGE_NAME"

echo "Copying files to package directory..."
cp dist/ax206monitor-linux-amd64 "$PACKAGE_NAME/ax206monitor"
cp install.sh "$PACKAGE_NAME/"
cp README.md "$PACKAGE_NAME/"

chmod +x "$PACKAGE_NAME/ax206monitor"
chmod +x "$PACKAGE_NAME/install.sh"

echo "Creating tar archive..."
tar -czf "dist/$PACKAGE_NAME.tar.gz" "$PACKAGE_NAME"

echo "Cleaning up package directory..."
rm -rf "$PACKAGE_NAME"

echo ""
echo "Package created successfully!"
echo "Output: dist/$PACKAGE_NAME.tar.gz"
echo ""
echo "Installation instructions:"
echo "1. Extract: tar -xzf $PACKAGE_NAME.tar.gz"
echo "2. Enter directory: cd $PACKAGE_NAME"
echo "3. Install as root: sudo ./install.sh"
echo ""
echo "Package contents:"
echo "- ax206monitor (binary)"
echo "- install.sh (installation script)"
echo "- README.md (documentation)"
echo ""
echo "File sizes:"
ls -lh "dist/$PACKAGE_NAME.tar.gz"
ls -lh dist/ax206monitor-linux-amd64 