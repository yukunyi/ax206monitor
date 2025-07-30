# AX206 Monitor

A lightweight system monitoring tool for AX206 USB displays and file output.

## Features

- Real-time system monitoring (CPU, Memory, GPU, Network, Temperature)
- Multiple display layouts optimized for different screen sizes
- Support for AX206 USB displays and PNG file output
- Systemd service integration for automatic startup

## Quick Installation

```bash
# Download and extract the latest release
tar -xzf ax206monitor-linux-amd64-v1.0.0.tar.gz
cd ax206monitor-linux-amd64-v1.0.0

# Install as system service
sudo ./install.sh install
```

## Available Layouts

- **mini.json** - Minimal layout (480x320) - Default
- **small.json** - Compact layout (480x320)
- **normal.json** - Standard layout (480x320)
- **full.json** - Complete layout (800x480)

## Service Management

```bash
sudo systemctl status ax206monitor    # Check status
sudo systemctl restart ax206monitor   # Restart service
journalctl -u ax206monitor -f         # View logs
```

## Configuration

Configuration files are located in `/etc/ax206monitor/samples/`. To change layout:

```bash
sudo ln -sf /etc/ax206monitor/samples/normal.json /etc/ax206monitor/default.json
sudo systemctl restart ax206monitor
```

## Building from Source

```bash
git clone https://github.com/yukunyi/ax206monitor.git
cd ax206monitor
./build.sh
sudo ./install.sh install
```

## Repository

https://github.com/yukunyi/ax206monitor
