# AX206 System Monitor

A system monitoring tool for AX206 USB mini display, providing real-time monitoring of CPU, GPU, memory, and network status. Written in Go, Linux only.

Based on this project: https://github.com/plumbum/go2dpf

## Features

- **Real-time System Monitoring**: Display CPU usage, temperature, frequency, memory usage, GPU status, and network speeds
- **Professional Interface**: Clean 4-panel layout with circular progress indicators
- **Font Caching**: Optimized performance with global font caching
- **Systemd Integration**: Easy installation and management as a system service

## Requirements

- Linux system
- Go 1.19 or later
- AX206 USB display device (GEMBIRD Digital Photo Frame)
- USB access permissions

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/ax206monitor.git
cd ax206monitor

# Install as system service
sudo ./install.sh
```

## Usage

### Service Management

```bash
# Check service status
sudo systemctl status ax206monitor

# View logs
sudo journalctl -u ax206monitor -f

# Stop service
sudo systemctl stop ax206monitor

# Start service
sudo systemctl start ax206monitor

# Restart service
sudo systemctl restart ax206monitor
```

### Manual Run

```bash
# Run directly (requires root for USB access)
sudo ./dist/ax206monitor-linux-amd64
```

### Uninstall

```bash
# Remove service and binary
sudo ./install.sh uninstall
```

## Display Layout

The monitor displays information in a 4-panel layout:

1. **CPU Panel**: Shows CPU usage, temperature, frequency (min/max), and load average
2. **GPU Panel**: Shows GPU usage, temperature, frequency, and fan speed (if discrete GPU detected)
3. **Memory Panel**: Shows memory usage percentage and total/used memory
4. **Network Panel**: Shows upload/download speeds and interface name

Additional information is shown in:
- **Header**: IP address and current time
- **Footer**: Fan speeds for all detected system fans

## Development

### Building

```bash
# Build for Linux
./build.sh
```

### Project Structure

```
ax206monitor/
├── src/ax206monitor/     # Source code
│   ├── main.go           # Main application
│   ├── display.go        # Display management
│   ├── monitor.go        # System monitoring
│   ├── dpf.go           # USB device communication
│   └── image.go         # Image processing
├── dist/                 # Compiled binaries
├── build.sh             # Build script
├── install.sh           # Installation script
└── README.md           # This file
```

## Troubleshooting

### USB Device Not Found

```bash
# Check if device is connected
lsusb | grep -i gembird

# Check USB permissions
ls -la /dev/bus/usb/001/023
```

### Service Not Starting

```bash
# Check service logs
sudo journalctl -u ax206monitor -n 50

# Reinstall service
sudo ./install.sh uninstall
sudo ./install.sh
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
