#!/bin/bash

set -e

VERSION="1.0.0"
LINUX_PACKAGE="ax206monitor-linux-amd64-v${VERSION}"
WINDOWS_PACKAGE="ax206monitor-windows-amd64-v${VERSION}"
DIST_DIR="dist"
CONFIG_DIR="config"

echo "AX206 System Monitor - Package Script"
echo "====================================="
echo "Version: $VERSION"
echo "Linux Package: $LINUX_PACKAGE"
echo "Windows Package: $WINDOWS_PACKAGE"
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
required_configs=("mini.json" "normal.json" "full.json")
for config in "${required_configs[@]}"; do
    if [ ! -f "$CONFIG_DIR/$config" ]; then
        echo "Error: Required configuration file '$config' not found in $CONFIG_DIR"
        exit 1
    fi
done

echo "Found configuration files:"
ls -la "$CONFIG_DIR"/*.json

cd src/ax206monitor

echo "Compiling Linux version..."
GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
    -trimpath \
    -buildmode=exe \
    -o ../../dist/ax206monitor-linux-amd64 .

echo "Compiling Windows version..."
GOOS=windows GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
    -trimpath \
    -o ../../dist/ax206monitor-windows-amd64.exe .

cd ../..

chmod +x dist/ax206monitor-linux-amd64

echo "Creating Linux package directory..."
rm -rf "$LINUX_PACKAGE"
mkdir -p "$LINUX_PACKAGE"

echo "Copying files to Linux package directory..."
cp dist/ax206monitor-linux-amd64 "$LINUX_PACKAGE/ax206monitor"
cp install.sh "$LINUX_PACKAGE/"
cp -r config "$LINUX_PACKAGE/"
if [ -f README.md ]; then
    cp README.md "$LINUX_PACKAGE/"
fi

chmod +x "$LINUX_PACKAGE/ax206monitor"
chmod +x "$LINUX_PACKAGE/install.sh"

echo "Creating Windows package directory..."
rm -rf "$WINDOWS_PACKAGE"
mkdir -p "$WINDOWS_PACKAGE"

echo "Copying files to Windows package directory..."
cp dist/ax206monitor-windows-amd64.exe "$WINDOWS_PACKAGE/ax206monitor.exe"
cp -r config "$WINDOWS_PACKAGE/"
cp config/windows.json "$WINDOWS_PACKAGE/config/default.json"
if [ -f README.md ]; then
    cp README.md "$WINDOWS_PACKAGE/"
fi
if [ -f docs/LIBRE_HARDWARE_MONITOR.md ]; then
    cp docs/LIBRE_HARDWARE_MONITOR.md "$WINDOWS_PACKAGE/"
fi

cat > "$WINDOWS_PACKAGE/start.bat" << 'EOF'
@echo off
cd /d "%~dp0"
ax206monitor.exe
pause
EOF

echo "Verifying Linux package contents..."
echo "Linux package directory contents:"
ls -la "$LINUX_PACKAGE/"
echo "Configuration files:"
ls -la "$LINUX_PACKAGE/config/"

echo "Verifying Windows package contents..."
echo "Windows package directory contents:"
ls -la "$WINDOWS_PACKAGE/"
echo "Configuration files:"
ls -la "$WINDOWS_PACKAGE/config/"

echo "Creating Linux tar archive..."
tar -czf "dist/$LINUX_PACKAGE.tar.gz" "$LINUX_PACKAGE"

echo "Creating Windows zip archive..."
if command -v zip &> /dev/null; then
    zip -r "dist/$WINDOWS_PACKAGE.zip" "$WINDOWS_PACKAGE"
else
    echo "Warning: zip command not found, creating tar archive instead"
    tar -czf "dist/$WINDOWS_PACKAGE.tar.gz" "$WINDOWS_PACKAGE"
fi

echo "Cleaning up package directories..."
rm -rf "$LINUX_PACKAGE" "$WINDOWS_PACKAGE"

echo ""
echo "Packages created successfully!"
echo "Output files:"
ls -la dist/*.tar.gz dist/*.zip 2>/dev/null || ls -la dist/*.tar.gz

echo ""
echo "Installation Instructions:"
echo ""
echo "Linux:"
echo "1. Extract: tar -xzf dist/$LINUX_PACKAGE.tar.gz"
echo "2. Enter directory: cd $LINUX_PACKAGE"
echo "3. Install as root: sudo ./install.sh"
echo "4. Run: ax206monitor -config <config_name>"
echo ""
echo "Windows:"
echo "1. Extract: dist/$WINDOWS_PACKAGE.zip (or .tar.gz)"
echo "2. Double-click start.bat or ax206monitor.exe"
echo "3. Configure Libre Hardware Monitor URL in config/default.json"
echo ""
echo "Available configurations:"
ls config/*.json | sed 's/config\///g' | sed 's/\.json//g' | sed 's/^/  /'

echo ""
echo "Note: Ensure AX206 device is connected with proper USB permissions before running"