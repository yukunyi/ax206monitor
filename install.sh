#!/bin/bash

set -e

SERVICE_NAME="ax206monitor"
INSTALL_DIR="/usr/local/bin"
SERVICE_DIR="/etc/systemd/system"
VERSION="1.0.0"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo "This script must be run as root (use sudo)"
        exit 1
    fi
}

build_binary() {
    if [ ! -f "dist/ax206monitor-linux-amd64" ]; then
        print_status "Binary not found, building with build.sh..."
        if [ ! -f "build.sh" ]; then
            echo "Error: build.sh not found"
            exit 1
        fi
        chmod +x build.sh
        ./build.sh
    fi
    
    if [ ! -f "dist/ax206monitor-linux-amd64" ]; then
        echo "Error: Failed to build binary"
        exit 1
    fi
    
    print_success "Binary built successfully"
}

install_binary() {
    print_status "Installing binary to $INSTALL_DIR/$SERVICE_NAME"
    cp "dist/ax206monitor-linux-amd64" "$INSTALL_DIR/$SERVICE_NAME"
    chmod 755 "$INSTALL_DIR/$SERVICE_NAME"
    print_success "Installed binary to $INSTALL_DIR/$SERVICE_NAME"
}

create_systemd_service() {
    print_status "Creating systemd service: $SERVICE_DIR/$SERVICE_NAME.service"
    
    cat > "$SERVICE_DIR/$SERVICE_NAME.service" << EOF
[Unit]
Description=AX206 System Monitor v$VERSION
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/$SERVICE_NAME
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    print_success "Created systemd service file"
}

enable_service() {
    print_status "Reloading systemd daemon"
    systemctl daemon-reload
    
    print_status "Enabling $SERVICE_NAME service"
    systemctl enable "$SERVICE_NAME"
    
    print_status "Starting $SERVICE_NAME service"
    systemctl start "$SERVICE_NAME"
    
    print_success "Service $SERVICE_NAME is now running"
}

show_status() {
    print_status "Service status:"
    systemctl status "$SERVICE_NAME" --no-pager -l
}

print_usage() {
    echo ""
    print_success "Installation completed successfully!"
    echo ""
    echo "Useful commands:"
    echo "  • Check service status:    sudo systemctl status $SERVICE_NAME"
    echo "  • View logs:              sudo journalctl -u $SERVICE_NAME -f"
    echo "  • Stop service:           sudo systemctl stop $SERVICE_NAME"
    echo "  • Start service:          sudo systemctl start $SERVICE_NAME"
    echo "  • Restart service:        sudo systemctl restart $SERVICE_NAME"
    echo "  • Disable auto-start:     sudo systemctl disable $SERVICE_NAME"
    echo ""
    echo "Configuration:"
    echo "  • Binary location:        $INSTALL_DIR/$SERVICE_NAME"
    echo "  • Service file:           $SERVICE_DIR/$SERVICE_NAME.service"
    echo "  • Version:                $VERSION"
}

uninstall() {
    print_status "Uninstalling AX206 System Monitor v$VERSION..."
    
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    
    rm -f "$SERVICE_DIR/$SERVICE_NAME.service"
    
    rm -f "$INSTALL_DIR/$SERVICE_NAME"
    
    systemctl daemon-reload
    
    print_success "Uninstallation completed"
}

show_version() {
    echo "AX206 System Monitor v$VERSION"
    exit 0
}

main() {
    echo "=================================================="
    echo "  AX206 System Monitor Installation Script"
    echo "  Version: $VERSION"
    echo "=================================================="
    echo ""
    
    if [ "$1" = "uninstall" ]; then
        uninstall
        exit 0
    fi
    
    if [ "$1" = "version" ] || [ "$1" = "--version" ] || [ "$1" = "-v" ]; then
        show_version
    fi
    
    check_root
    build_binary
    
    print_status "Starting installation process..."
    echo ""
    
    install_binary
    create_systemd_service
    enable_service
    
    echo ""
    show_status
    
    echo ""
    print_usage
}

main "$@" 