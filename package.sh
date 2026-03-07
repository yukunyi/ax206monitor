#!/bin/bash

set -e

VERSION="1.0.0"
LINUX_PACKAGE="ax206monitor-linux-amd64-v${VERSION}"
WINDOWS_PACKAGE="ax206monitor-windows-amd64-v${VERSION}"
DIST_DIR="dist"
FRONTEND_DIR="frontend"
EMBED_DIST_DIR="src/ax206monitor/webassets/webdist"

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

if ! command -v npm &> /dev/null; then
    echo "Error: npm not found, cannot build web frontend"
    exit 1
fi

echo "Building Vite frontend..."
pushd "$FRONTEND_DIR" > /dev/null
npm install
npm run build
popd > /dev/null

if [ ! -f "$FRONTEND_DIR/dist/index.html" ]; then
    echo "Error: frontend build output missing: $FRONTEND_DIR/dist/index.html"
    exit 1
fi

echo "Syncing frontend dist to $EMBED_DIST_DIR ..."
mkdir -p "$EMBED_DIST_DIR"
rm -rf "$EMBED_DIST_DIR"/*
cp -r "$FRONTEND_DIR"/dist/* "$EMBED_DIST_DIR"/

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

cd ../../src/ax206monitor

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
if [ -f README.md ]; then
    cp README.md "$LINUX_PACKAGE/"
fi

chmod +x "$LINUX_PACKAGE/ax206monitor"

echo "Creating Windows package directory..."
rm -rf "$WINDOWS_PACKAGE"
mkdir -p "$WINDOWS_PACKAGE"

echo "Copying files to Windows package directory..."
cp dist/ax206monitor-windows-amd64.exe "$WINDOWS_PACKAGE/ax206monitor.exe"
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

echo "Verifying Windows package contents..."
echo "Windows package directory contents:"
ls -la "$WINDOWS_PACKAGE/"

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
echo "3. Install as root service: sudo ./ax206monitor --install"
echo "4. Install as user service: ./ax206monitor --install"
echo "5. Run in foreground: ./ax206monitor"
echo ""
echo "Windows:"
echo "1. Extract: dist/$WINDOWS_PACKAGE.zip (or .tar.gz)"
echo "2. Double-click start.bat or ax206monitor.exe"
echo "3. Use Web UI if needed: set AX206_MONITOR_WEB=1 && ax206monitor.exe --port 18086"

echo ""
echo "Note: Ensure AX206 device is connected with proper USB permissions before running"
