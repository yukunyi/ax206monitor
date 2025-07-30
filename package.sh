#!/bin/bash

set -e

VERSION="1.0.0"
PACKAGE_NAME="ax206monitor-linux-amd64-v${VERSION}"
DIST_DIR="dist"
CONFIG_DIR="config"

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

echo "Validating configuration files..."
cd ../../
if [ ! -d "$CONFIG_DIR" ]; then
    echo "Error: Configuration directory '$CONFIG_DIR' not found"
    exit 1
fi

# Check for required configuration files
required_configs=("mini.json" "small.json" "normal.json" "full.json")
for config in "${required_configs[@]}"; do
    if [ ! -f "$CONFIG_DIR/$config" ]; then
        echo "Error: Required configuration file '$config' not found in $CONFIG_DIR"
        exit 1
    fi
done

echo "Found configuration files:"
ls -la "$CONFIG_DIR"/*.json

cd src/ax206monitor

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
cp -r config "$PACKAGE_NAME/"
if [ -f README.md ]; then
    cp README.md "$PACKAGE_NAME/"
fi

chmod +x "$PACKAGE_NAME/ax206monitor"
chmod +x "$PACKAGE_NAME/install.sh"

echo "Verifying package contents..."
echo "Package directory contents:"
ls -la "$PACKAGE_NAME/"
echo "Configuration files:"
ls -la "$PACKAGE_NAME/config/"

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
echo "- config/ (configuration files)"
echo "  - mini.json (minimal layout)"
echo "  - small.json (compact layout)"
echo "  - normal.json (standard layout)"
echo "  - full.json (complete layout)"
if [ -f "$PACKAGE_NAME/README.md" ]; then
    echo "- README.md (documentation)"
fi
echo ""
echo "File sizes:"
ls -lh "dist/$PACKAGE_NAME.tar.gz"
ls -lh dist/ax206monitor-linux-amd64 