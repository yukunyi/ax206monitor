package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gopsutilDisk "github.com/shirou/gopsutil/v3/disk"
)

// DiskIOSnapshot represents a snapshot of disk I/O statistics
type DiskIOSnapshot struct {
	ReadSectors  int64
	WriteSectors int64
	Timestamp    time.Time
}

var (
	cpuTempSensor      = NewCachedSensorPath(30 * time.Second)
	lastDiskStats      map[string]*DiskIOSnapshot
	cachedDiskTempPath string
	diskStatsMu        sync.Mutex
	diskTempPathMu     sync.Mutex
	linuxNetStatsMu    sync.Mutex

	// per-device disk temperature sensor path cache with TTL
	diskTempCacheMu sync.Mutex
	diskTempCache   = make(map[string]struct {
		path string
		last time.Time
	})
	diskTempCacheTTL = 30 * time.Second
)

func getLinuxCPUTemperature() float64 {
	temp, err := cpuTempSensor.GetValue(CPUSensorPatterns, "temp1_input", CPUTempMin, CPUTempMax)
	if err != nil {
		logDebugModule("cpu", "CPU temperature not available: %v", err)
		return 0.0
	}
	return temp
}

func getLinuxDiskTemperature() float64 {
	diskTempPathMu.Lock()
	path := cachedDiskTempPath
	diskTempPathMu.Unlock()

	if path != "" {
		if tempBytes, err := os.ReadFile(path); err == nil {
			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 0 && temp < 100 {
					return temp
				}
			}
		}
		diskTempPathMu.Lock()
		cachedDiskTempPath = ""
		diskTempPathMu.Unlock()
		logInfoModule("disk", "Disk temperature path changed, rescanning")
	}

	hwmonDirs, err := os.ReadDir("/sys/class/hwmon")
	if err != nil {
		return 0.0
	}

	for _, hwmon := range hwmonDirs {
		hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", hwmon.Name())

		nameBytes, err := os.ReadFile(hwmonPath + "/name")
		if err != nil {
			continue
		}
		hwmonName := strings.TrimSpace(string(nameBytes))

		isDiskSensor := false
		if strings.Contains(hwmonName, "nvme") ||
			strings.Contains(hwmonName, "sata") ||
			strings.Contains(hwmonName, "ata") ||
			strings.Contains(hwmonName, "scsi") ||
			hwmonName == "drivetemp" {
			isDiskSensor = true
		}

		if isDiskSensor {
			tempPath := hwmonPath + "/temp1_input"
			tempBytes, err := os.ReadFile(tempPath)
			if err != nil {
				continue
			}

			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 0 && temp < 100 {
					diskTempPathMu.Lock()
					cachedDiskTempPath = tempPath
					diskTempPathMu.Unlock()
					logInfoModule("disk", "Found disk temperature sensor: %s (%.1f°C)", hwmonName, temp)
					return temp
				}
			}
		}
	}
	return 0.0
}

// getRealCPUFrequency gets real CPU frequency (min and max) (Linux)
func getLinuxCPUFrequency() (float64, float64) {
	// Try to read max frequency from /sys
	maxFreqBytes, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")
	if err == nil {
		maxFreqStr := strings.TrimSpace(string(maxFreqBytes))
		if maxFreqInt, err := strconv.Atoi(maxFreqStr); err == nil {
			maxFreq := float64(maxFreqInt) / 1000.0 // Convert from kHz to MHz

			// Try to read current frequency
			curFreqBytes, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq")
			if err == nil {
				curFreqStr := strings.TrimSpace(string(curFreqBytes))
				if curFreqInt, err := strconv.Atoi(curFreqStr); err == nil {
					curFreq := float64(curFreqInt) / 1000.0 // Convert from kHz to MHz
					return curFreq, maxFreq
				}
			}

			return maxFreq, maxFreq
		}
	}

	return 0.0, 0.0
}

var (
	lastNetTime  time.Time
	lastNetStats map[string]uint64
)

func getLinuxNetworkInfo() NetworkInfoData {
	info := NetworkInfoData{}

	// Get IP address
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				addrs, err := iface.Addrs()
				if err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
							if ipnet.IP.To4() != nil {
								info.IP = ipnet.IP.String()
								break
							}
						}
					}
				}
				if info.IP != "" {
					break
				}
			}
		}
	}

	// Get network speeds
	now := time.Now()
	linuxNetStatsMu.Lock()
	defer linuxNetStatsMu.Unlock()
	if !lastNetTime.IsZero() {
		duration := now.Sub(lastNetTime).Seconds()
		if duration > 0 {
			statsPath := "/proc/net/dev"
			data, err := os.ReadFile(statsPath)
			if err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					if strings.Contains(line, ":") && !strings.Contains(line, "lo:") {
						parts := strings.Fields(line)
						if len(parts) >= 10 {
							rxBytes, _ := strconv.ParseUint(parts[1], 10, 64)
							txBytes, _ := strconv.ParseUint(parts[9], 10, 64)

							if lastNetStats != nil {
								if lastRx, exists := lastNetStats["rx"]; exists && rxBytes > lastRx {
									info.DownloadSpeed = float64(rxBytes-lastRx) / duration / 1024 / 1024
								}
								if lastTx, exists := lastNetStats["tx"]; exists && txBytes > lastTx {
									info.UploadSpeed = float64(txBytes-lastTx) / duration / 1024 / 1024
								}
							}

							if lastNetStats == nil {
								lastNetStats = make(map[string]uint64)
							}
							lastNetStats["rx"] = rxBytes
							lastNetStats["tx"] = txBytes
							break
						}
					}
				}
			}
		}
	}

	lastNetTime = now
	return info
}

// detectLinuxCPUInfo detects detailed CPU information on Linux
func detectLinuxCPUInfo() *CPUInfo {
	cpuInfo := &CPUInfo{
		Model:        "Unknown CPU",
		Cores:        1,
		Threads:      1,
		Architecture: "unknown",
		MaxFreq:      0,
		MinFreq:      0,
		Vendor:       "unknown",
	}

	// Read /proc/cpuinfo
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		coreCount := 0

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "model name":
				if cpuInfo.Model == "Unknown CPU" {
					// Clean up the model name
					model := value
					model = strings.ReplaceAll(model, "(R)", "")
					model = strings.ReplaceAll(model, "(TM)", "")
					model = strings.ReplaceAll(model, "  ", " ")
					cpuInfo.Model = strings.TrimSpace(model)
				}
			case "vendor_id":
				if cpuInfo.Vendor == "unknown" {
					switch value {
					case "GenuineIntel":
						cpuInfo.Vendor = "Intel"
					case "AuthenticAMD":
						cpuInfo.Vendor = "AMD"
					default:
						cpuInfo.Vendor = value
					}
				}
			case "processor":
				coreCount++
			case "cpu MHz":
				if freq, err := strconv.ParseFloat(value, 64); err == nil {
					if cpuInfo.MaxFreq == 0 || freq > cpuInfo.MaxFreq {
						cpuInfo.MaxFreq = freq
					}
					if cpuInfo.MinFreq == 0 || freq < cpuInfo.MinFreq {
						cpuInfo.MinFreq = freq
					}
				}
			}
		}

		cpuInfo.Threads = coreCount
	}

	// Try to get core count from /sys/devices/system/cpu/
	if entries, err := os.ReadDir("/sys/devices/system/cpu/"); err == nil {
		coreCount := 0
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "cpu") && len(entry.Name()) > 3 {
				if _, err := strconv.Atoi(entry.Name()[3:]); err == nil {
					coreCount++
				}
			}
		}
		if coreCount > 0 {
			cpuInfo.Cores = coreCount
		}
	}

	// Get architecture
	cpuInfo.Architecture = runtime.GOARCH

	// Try to get frequency limits from cpufreq
	if data, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"); err == nil {
		if freq, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			cpuInfo.MaxFreq = freq / 1000 // Convert from kHz to MHz
		}
	}

	if data, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_min_freq"); err == nil {
		if freq, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			cpuInfo.MinFreq = freq / 1000 // Convert from kHz to MHz
		}
	}

	return cpuInfo
}

// detectLinuxDiskInfo detects detailed disk information on Linux
func detectLinuxDiskInfo() []*DiskInfo {
	var disks []*DiskInfo
	mountsByDevice := getMountPointsByDevice()

	// Read block devices from /sys/block
	if entries, err := os.ReadDir("/sys/block"); err == nil {
		for _, entry := range entries {
			deviceName := strings.TrimSpace(entry.Name())
			if deviceName == "" {
				continue
			}

			// Skip virtual devices, loop devices, ram disks, etc.
			if strings.HasPrefix(deviceName, "loop") ||
				strings.HasPrefix(deviceName, "ram") ||
				strings.HasPrefix(deviceName, "dm-") ||
				strings.HasPrefix(deviceName, "zram") ||
				strings.HasPrefix(deviceName, "md") ||
				strings.HasPrefix(deviceName, "sr") ||
				strings.HasPrefix(deviceName, "fd") {
				continue
			}

			// Only include physical storage devices (nvme, sd, hd)
			if !strings.HasPrefix(deviceName, "nvme") &&
				!strings.HasPrefix(deviceName, "sd") &&
				!strings.HasPrefix(deviceName, "hd") {
				continue
			}

			diskPath := fmt.Sprintf("/sys/block/%s", deviceName)
			disk := &DiskInfo{
				Name:        deviceName,
				Model:       "Unknown",
				Size:        0,
				Temperature: 0,
				ReadSpeed:   0,
				WriteSpeed:  0,
				Usage:       0,
			}

			// Get disk model
			if modelData, err := os.ReadFile(diskPath + "/device/model"); err == nil {
				disk.Model = strings.TrimSpace(string(modelData))
			}

			// Get disk size (in sectors, multiply by 512 to get bytes)
			if sizeData, err := os.ReadFile(diskPath + "/size"); err == nil {
				if sectors, err := strconv.ParseInt(strings.TrimSpace(string(sizeData)), 10, 64); err == nil {
					disk.Size = (sectors * 512) / (1024 * 1024 * 1024) // Convert to GB
				}
			}

			// Try to get disk temperature from hwmon
			disk.Temperature = getDiskTemperatureByName(deviceName)

			// Get disk I/O stats and calculate real-time speeds
			if statData, err := os.ReadFile(diskPath + "/stat"); err == nil {
				fields := strings.Fields(string(statData))
				if len(fields) >= 10 {
					// Parse current stats
					var currentReadSectors, currentWriteSectors int64
					if sectorsRead, err := strconv.ParseInt(fields[2], 10, 64); err == nil {
						currentReadSectors = sectorsRead
					}
					if sectorsWritten, err := strconv.ParseInt(fields[6], 10, 64); err == nil {
						currentWriteSectors = sectorsWritten
					}

					// Calculate speed based on previous measurement
					now := time.Now()
					diskStatsMu.Lock()
					if lastStats, exists := lastDiskStats[deviceName]; exists {
						timeDiff := now.Sub(lastStats.Timestamp).Seconds()
						if timeDiff > 0 {
							readDiff := currentReadSectors - lastStats.ReadSectors
							writeDiff := currentWriteSectors - lastStats.WriteSectors

							// Convert sectors to MB/s (512 bytes per sector)
							disk.ReadSpeed = float64(readDiff) * 512 / (1024 * 1024) / timeDiff
							disk.WriteSpeed = float64(writeDiff) * 512 / (1024 * 1024) / timeDiff
						}
					}

					// Store current stats for next calculation
					if lastDiskStats == nil {
						lastDiskStats = make(map[string]*DiskIOSnapshot)
					}
					lastDiskStats[deviceName] = &DiskIOSnapshot{
						ReadSectors:  currentReadSectors,
						WriteSectors: currentWriteSectors,
						Timestamp:    now,
					}
					diskStatsMu.Unlock()
				}
			}

			disk.Usage = getDiskUsagePercentage(deviceName, mountsByDevice)

			disks = append(disks, disk)
		}
	}

	return disks
}

// getDiskTemperatureByName tries to get disk temperature by device name
func getDiskTemperatureByName(deviceName string) float64 {
	// 1) try cached path first (per device)
	diskTempCacheMu.Lock()
	entry, ok := diskTempCache[deviceName]
	if ok && time.Since(entry.last) < diskTempCacheTTL && entry.path != "" {
		p := entry.path
		diskTempCacheMu.Unlock()
		if tempData, err := os.ReadFile(p); err == nil {
			if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64); err == nil {
				t := temp / 1000.0
				if t > 0 && t < 100 {
					return t
				}
			}
		}
		// fallthrough to rescan if cache invalid
	} else {
		diskTempCacheMu.Unlock()
	}

	// Method 1: Try to find temperature in hwmon for this specific disk
	if hwmonEntries, err := os.ReadDir("/sys/class/hwmon"); err == nil {
		for _, entry := range hwmonEntries {
			hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", entry.Name())

			// Check if this hwmon is for our disk
			if nameData, err := os.ReadFile(hwmonPath + "/name"); err == nil {
				name := strings.TrimSpace(string(nameData))

				// Check for disk-specific hwmon names
				if strings.Contains(strings.ToLower(name), strings.ToLower(deviceName)) ||
					strings.Contains(strings.ToLower(name), "drivetemp") ||
					(strings.Contains(strings.ToLower(name), "nvme") && strings.Contains(deviceName, "nvme")) ||
					(strings.Contains(strings.ToLower(name), "ata") && strings.HasPrefix(deviceName, "sd")) {

					// Look for temperature sensors
					if tempEntries, err := os.ReadDir(hwmonPath); err == nil {
						for _, tempEntry := range tempEntries {
							if strings.HasPrefix(tempEntry.Name(), "temp") && strings.HasSuffix(tempEntry.Name(), "_input") {
								tempPath := fmt.Sprintf("%s/%s", hwmonPath, tempEntry.Name())
								if tempData, err := os.ReadFile(tempPath); err == nil {
									if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64); err == nil {
										tempCelsius := temp / 1000.0              // Convert from millidegrees to degrees
										if tempCelsius > 0 && tempCelsius < 100 { // Sanity check
											logDebugModule("disk", "Found temperature for %s via hwmon %s: %.1f°C", deviceName, name, tempCelsius)
											// cache sensor path
											diskTempCacheMu.Lock()
											diskTempCache[deviceName] = struct {
												path string
												last time.Time
											}{path: tempPath, last: time.Now()}
											diskTempCacheMu.Unlock()
											return tempCelsius
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Method 2: Try alternative method via /sys/block/*/device/hwmon/*/temp*_input
	blockPath := fmt.Sprintf("/sys/block/%s/device/hwmon", deviceName)
	if hwmonEntries, err := os.ReadDir(blockPath); err == nil {
		for _, entry := range hwmonEntries {
			hwmonPath := fmt.Sprintf("%s/%s", blockPath, entry.Name())
			if tempEntries, err := os.ReadDir(hwmonPath); err == nil {
				for _, tempEntry := range tempEntries {
					if strings.HasPrefix(tempEntry.Name(), "temp") && strings.HasSuffix(tempEntry.Name(), "_input") {
						tempPath := fmt.Sprintf("%s/%s", hwmonPath, tempEntry.Name())
						if tempData, err := os.ReadFile(tempPath); err == nil {
							if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64); err == nil {
								tempCelsius := temp / 1000.0
								if tempCelsius > 0 && tempCelsius < 100 {
									logDebugModule("disk", "Found temperature for %s via device path: %.1f°C", deviceName, tempCelsius)
									// cache sensor path
									diskTempCacheMu.Lock()
									diskTempCache[deviceName] = struct {
										path string
										last time.Time
									}{path: tempPath, last: time.Now()}
									diskTempCacheMu.Unlock()
									return tempCelsius
								}
							}
						}
					}
				}
			}
		}
	}

	// Method 3: For NVMe drives, try /sys/class/nvme/nvme*/hwmon*/temp*_input
	if strings.HasPrefix(deviceName, "nvme") {
		controller := nvmeControllerName(deviceName)
		if controller == "" {
			return 0.0
		}
		nvmePath := fmt.Sprintf("/sys/class/nvme/%s", controller)
		if nvmeEntries, err := os.ReadDir(nvmePath); err == nil {
			for _, entry := range nvmeEntries {
				if strings.HasPrefix(entry.Name(), "hwmon") {
					hwmonPath := fmt.Sprintf("%s/%s", nvmePath, entry.Name())
					if tempEntries, err := os.ReadDir(hwmonPath); err == nil {
						for _, tempEntry := range tempEntries {
							if strings.HasPrefix(tempEntry.Name(), "temp") && strings.HasSuffix(tempEntry.Name(), "_input") {
								tempPath := fmt.Sprintf("%s/%s", hwmonPath, tempEntry.Name())
								if tempData, err := os.ReadFile(tempPath); err == nil {
									if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64); err == nil {
										tempCelsius := temp / 1000.0
										if tempCelsius > 0 && tempCelsius < 100 {
											logDebugModule("disk", "Found temperature for %s via nvme path: %.1f°C", deviceName, tempCelsius)
											// cache sensor path
											diskTempCacheMu.Lock()
											diskTempCache[deviceName] = struct {
												path string
												last time.Time
											}{path: tempPath, last: time.Now()}
											diskTempCacheMu.Unlock()
											return tempCelsius
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	logDebugModule("disk", "No temperature sensor found for disk %s", deviceName)
	return 0.0 // No temperature found
}

func nvmeControllerName(deviceName string) string {
	name := strings.TrimSpace(deviceName)
	if !strings.HasPrefix(name, "nvme") {
		return ""
	}
	rest := strings.TrimPrefix(name, "nvme")
	if rest == "" {
		return ""
	}
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return ""
	}
	return "nvme" + rest[:end]
}

// getDiskUsagePercentage calculates disk usage percentage for a device
func getMountPointsByDevice() map[string][]string {
	mountsByDevice := make(map[string][]string)
	mountsData, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return mountsByDevice
	}

	lines := strings.Split(string(mountsData), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			device := fields[0]
			mountPoint := fields[1]
			if strings.HasPrefix(device, "/dev/") {
				addMountPoint(mountsByDevice, strings.TrimPrefix(device, "/dev/"), mountPoint)
			}
		}
	}
	return mountsByDevice
}

func addMountPoint(mountsByDevice map[string][]string, deviceName, mountPoint string) {
	deviceName = strings.TrimSpace(deviceName)
	mountPoint = strings.TrimSpace(mountPoint)
	if deviceName == "" || mountPoint == "" {
		return
	}
	points := mountsByDevice[deviceName]
	for _, existing := range points {
		if existing == mountPoint {
			return
		}
	}
	mountsByDevice[deviceName] = append(points, mountPoint)
}

// getDiskUsagePercentage calculates disk usage percentage for a device
func getDiskUsagePercentage(deviceName string, mountsByDevice map[string][]string) float64 {
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		return 0.0
	}
	candidates := make([]string, 0, 4)
	seenMounts := make(map[string]struct{})

	addCandidate := func(mountPoint string) {
		mountPoint = strings.TrimSpace(mountPoint)
		if mountPoint == "" {
			return
		}
		if _, exists := seenMounts[mountPoint]; exists {
			return
		}
		seenMounts[mountPoint] = struct{}{}
		candidates = append(candidates, mountPoint)
	}

	for _, mountPoint := range mountsByDevice[deviceName] {
		addCandidate(mountPoint)
	}
	for mountedDevice, points := range mountsByDevice {
		if mountedDevice == deviceName {
			continue
		}
		if strings.HasPrefix(mountedDevice, deviceName) || strings.Contains(mountedDevice, deviceName) {
			for _, mountPoint := range points {
				addCandidate(mountPoint)
			}
		}
	}

	if len(candidates) == 0 {
		return 0.0
	}
	sort.Strings(candidates)
	peakUsage := 0.0
	for _, mountPoint := range candidates {
		if usage, ok := getFilesystemUsage(mountPoint); ok && usage > peakUsage {
			peakUsage = usage
		}
	}
	if peakUsage > 0 {
		return peakUsage
	}

	return 0.0
}

// getFilesystemUsage gets filesystem usage percentage for a mount point
func getFilesystemUsage(mountPoint string) (float64, bool) {
	mountPoint = strings.TrimSpace(mountPoint)
	if mountPoint == "" {
		return 0, false
	}
	usage, err := gopsutilDisk.Usage(mountPoint)
	if err != nil || usage == nil {
		return 0, false
	}
	if usage.UsedPercent < 0 {
		return 0, false
	}
	return usage.UsedPercent, true
}
