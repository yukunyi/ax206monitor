#!/bin/bash

# Common build functions to reduce duplication between build.sh and package.sh

set -e

VERSION="${VERSION:-1.0.0}"

FRONTEND_DIR="frontend"
EMBED_DIST_DIR="src/metricsrendersender/webassets/webdist"
DIST_DIR="${DIST_DIR:-dist}"
BUILD_TARGETS="${BUILD_TARGETS:-linux windows}"
SKIP_FRONTEND_BUILD="${SKIP_FRONTEND_BUILD:-0}"
SKIP_GO_MOD_TIDY="${SKIP_GO_MOD_TIDY:-0}"
WINDOWS_CC="${WINDOWS_CC:-x86_64-w64-mingw32-gcc}"
WINDOWS_CXX="${WINDOWS_CXX:-x86_64-w64-mingw32-g++}"
WINDOWS_PKG_CONFIG="${WINDOWS_PKG_CONFIG:-x86_64-w64-mingw32-pkg-config}"

has_build_target() {
    local needle="$1"
    for target in $BUILD_TARGETS; do
        if [ "$target" = "$needle" ]; then
            return 0
        fi
    done
    return 1
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        echo "Error: Go compiler not found, please install Go first"
        exit 1
    fi
    echo "Go version: $(go version)"
}

# Initialize Go module if needed
init_go_module() {
    cd src/metricsrendersender
    if [ ! -f go.mod ]; then
        echo "Initializing Go module..."
        go mod init metricsrendersender
    fi
    if [ "$SKIP_GO_MOD_TIDY" = "1" ]; then
        echo "Skipping dependency tidy"
    else
        echo "Downloading dependencies..."
        go mod tidy
    fi
    cd ../..
}

# Build web frontend and sync assets to embedded directory
build_frontend_assets() {
    if [ ! -d "$FRONTEND_DIR" ]; then
        echo "Frontend directory '$FRONTEND_DIR' not found, skip web UI build"
        return
    fi

    if ! command -v npm &> /dev/null; then
        echo "Error: npm not found, cannot build web frontend"
        exit 1
    fi

    echo "Building Vite frontend..."
    pushd "$FRONTEND_DIR" > /dev/null
    if [ -f package-lock.json ]; then
        npm ci --include=dev
    else
        npm install --include=dev
    fi
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
}

# Clean previous build files
clean_dist() {
    echo "Cleaning previous build files..."
    mkdir -p "$DIST_DIR"
    rm -rf "$DIST_DIR"/*
}

# Compile for Linux
compile_linux() {
    echo "Compiling Linux version..."
    cd src/metricsrendersender
    GOOS=linux GOARCH=amd64 go build \
        -ldflags "-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        -trimpath \
        -buildmode=exe \
        -o ../../"$DIST_DIR"/metricsrendersender-linux-amd64 .
    cd ../..
    chmod +x "$DIST_DIR"/metricsrendersender-linux-amd64
}

# Compile for Windows
compile_windows() {
    echo "Compiling Windows version..."
    if ! command -v "$WINDOWS_CC" &> /dev/null; then
        echo "Error: $WINDOWS_CC not found, cannot compile windows with cgo"
        exit 1
    fi
    if ! command -v "$WINDOWS_CXX" &> /dev/null; then
        echo "Error: $WINDOWS_CXX not found, cannot compile windows with cgo"
        exit 1
    fi
    cd src/metricsrendersender
    CC="$WINDOWS_CC" \
    CXX="$WINDOWS_CXX" \
    PKG_CONFIG="$WINDOWS_PKG_CONFIG" \
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build \
        -ldflags "-s -w -H=windowsgui -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        -trimpath \
        -o ../../"$DIST_DIR"/metricsrendersender-windows-amd64.exe .
    cd ../..
}

# Validate configuration files
validate_configs() {
    CONFIG_DIR="config"
    echo "Validating configuration files..."
    
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
}

# Show build results
show_build_results() {
    echo ""
    echo "Compilation complete!"
    echo "Output files in $DIST_DIR directory:"
    ls -la "$DIST_DIR"/
    echo ""
    echo "Usage Instructions:"
    echo "Linux: ./$DIST_DIR/metricsrendersender-linux-amd64"
    echo "Windows: $DIST_DIR/metricsrendersender-windows-amd64.exe"
    echo ""
    echo "Note: Configure at least one output target before production use"
}

# Common build steps
common_build() {
    check_go
    clean_dist
    if [ "$SKIP_FRONTEND_BUILD" = "1" ]; then
        echo "Skipping frontend build (SKIP_FRONTEND_BUILD=1)"
    else
        build_frontend_assets
    fi
    init_go_module
    if has_build_target "linux"; then
        compile_linux
    fi
    if has_build_target "windows"; then
        compile_windows
    fi
}
