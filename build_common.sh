#!/bin/bash

# Common build functions to reduce duplication between build.sh and package.sh

set -e

VERSION="1.0.0"

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
    cd src/ax206monitor
    if [ ! -f go.mod ]; then
        echo "Initializing Go module..."
        go mod init ax206monitor
    fi
    echo "Downloading dependencies..."
    go mod tidy
    cd ../..
}

# Clean previous build files
clean_dist() {
    echo "Cleaning previous build files..."
    mkdir -p dist
    rm -rf dist/*
}

# Compile for Linux
compile_linux() {
    echo "Compiling Linux version..."
    cd src/ax206monitor
    GOOS=linux GOARCH=amd64 go build \
        -ldflags "-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        -trimpath \
        -buildmode=exe \
        -o ../../dist/ax206monitor-linux-amd64 .
    cd ../..
    chmod +x dist/ax206monitor-linux-amd64
}

# Compile for Windows
compile_windows() {
    echo "Compiling Windows version..."
    cd src/ax206monitor
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
        -ldflags "-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        -trimpath \
        -o ../../dist/ax206monitor-windows-amd64.exe .
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
    echo "Output files in dist/ directory:"
    ls -la dist/
    echo ""
    echo "Usage Instructions:"
    echo "Linux: ./dist/ax206monitor-linux-amd64"
    echo "Windows: dist/ax206monitor-windows-amd64.exe"
    echo ""
    echo "Note: Ensure AX206 device is connected with proper USB permissions before running"
}

# Common build steps
common_build() {
    check_go
    clean_dist
    init_go_module
    compile_linux
    compile_windows
}
