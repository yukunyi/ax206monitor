# AX206 Monitor

A lightweight system monitoring tool for AX206 USB displays and memimg output.

## Features

- Real-time system monitoring (CPU, Memory, GPU, Network, Temperature)
- Multiple display layouts optimized for different screen sizes
- Support for AX206 USB displays and memimg output
- Desktop tray integration (Linux / Windows)
- User auto-start toggle from tray menu

## Quick Installation

```bash
# Download and extract the latest release
tar -xzf ax206monitor-linux-amd64-v1.0.0.tar.gz
cd ax206monitor-linux-amd64-v1.0.0

# Run directly
./ax206monitor
```

## Windows Package Layout

After `./package.sh`, Windows release files are placed at:

```text
dist/windows/ax206_monitor/
├── ax206monitor.exe
├── README.md
└── (required DLL dependencies, e.g. libusb-1.0.dll)
```

`package.sh` also generates `dist/windows/ax206_monitor.zip`.

## Configuration

Runtime config path: `$HOME/.config/ax206monitor/config.json`.

Config directory layout:

```text
$HOME/.config/ax206monitor/
├── config.json
├── history/
└── profiles/
    ├── active-profile
    └── *.json
```

### Custom monitors

You can define custom monitor sources with `custom_monitors`:

- `file`: read value from a sysfs file (supports millidegree auto conversion)
- `mixed`: aggregate multiple monitors (`max` / `min` / `avg`)
- `coolercontrol`: read sensor values from CoolerControl SSE stream (`/sse/status`, with `/status` fallback)

```json
{
  "coolercontrol_url": "http://127.0.0.1:11987",
  "custom_monitors": [
    {
      "name": "chipset_temp",
      "label": "Chipset",
      "type": "file",
      "path": "/sys/class/hwmon/hwmon6/temp1_input"
    },
    {
      "name": "board_temp_max",
      "label": "Board Max",
      "type": "mixed",
      "sources": ["cpu_temp", "chipset_temp", "disk_default_temp"],
      "aggregate": "max"
    },
    {
      "name": "cc_gpu_temp",
      "label": "CC GPU",
      "type": "coolercontrol",
      "device_uid": "GPU-1234",
      "temp_name": "gpu_temp"
    }
  ]
}
```

## Web Configuration UI

AX206 Monitor now supports a web configuration backend (Echo) and Vite frontend editor.

### Start web UI

```bash
# Release mode: serve embedded frontend resources
AX206_MONITOR_WEB=1 ./dist/ax206monitor-linux-amd64 --port 18086

# Development mode: proxy frontend requests to Vite dev server
AX206_MONITOR_DEV_URL=http://127.0.0.1:18087 ./dist/ax206monitor-linux-amd64 --port 18086
```

Open: `http://127.0.0.1:18086`

Web UI features:
- Configure all top-level `MonitorConfig` fields (including `font_sizes`, `colors`, `labels`, `units`, `color_thresholds`)
- Configure `custom_monitors` (`file` / `mixed` / `coolercontrol`)
- Visual editor for `items` (add/clone, select, drag-move, resize, align, property editing, preview)
- Preview image is rendered by Go backend (`/api/preview`, PNG)
- Save config to `$HOME/.config/ax206monitor/config.json`
- Structured map editors for `colors`, `labels`, `units`, `color_thresholds`
- Import/export JSON and rollback to history versions (`~/.config/ax206monitor/history`)
- Multi-profile management (create/switch/save-as/delete profile)
  - Profiles directory: `~/.config/ax206monitor/profiles`
  - Active profile marker: `~/.config/ax206monitor/profiles/active-profile`
  - Switching profile updates `~/.config/ax206monitor/config.json`

### Frontend development

```bash
cd frontend
npm install
npm run dev
```

### Build embedding

`build.sh` / `package.sh` now automatically:
1. Build Vite frontend (`frontend/dist`)
2. Sync built assets to `src/ax206monitor/webassets/webdist`
3. Embed assets into Go binary via `go:embed`

## Building from Source

```bash
git clone https://github.com/yukunyi/ax206monitor.git
cd ax206monitor
./build.sh
./dist/ax206monitor-linux-amd64
```

## Repository

https://github.com/yukunyi/ax206monitor
