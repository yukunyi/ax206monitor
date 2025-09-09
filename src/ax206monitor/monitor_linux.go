//go:build linux

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DiskIOSnapshot represents a snapshot of disk I/O statistics
type DiskIOSnapshot struct {
	ReadSectors  int64
	WriteSectors int64
	Timestamp    time.Time
}

var (
	cpuTempSensor      = NewCachedSensorPath(30 * time.Second)
	gpuTempSensor      = NewCachedSensorPath(30 * time.Second)
	diskTempSensor     = NewCachedSensorPath(30 * time.Second)
	lastDiskStats      map[string]*DiskIOSnapshot
	cachedDiskTempPath string
	cachedGPUTempPath  string

	// per-device disk temperature sensor path cache with TTL
	diskTempCacheMu sync.Mutex
	diskTempCache   = make(map[string]struct {
		path string
		last time.Time
	})
	diskTempCacheTTL = 30 * time.Second
)

func getRealCPUTemperature() float64 {
	temp, err := cpuTempSensor.GetValue(CPUSensorPatterns, "temp1_input", CPUTempMin, CPUTempMax)
	if err != nil {
		logDebugModule("cpu", "CPU temperature not available: %v", err)
		return 0.0
	}
	return temp
}

func getDiskTemperature() float64 {
	if cachedDiskTempPath != "" {
		if tempBytes, err := ioutil.ReadFile(cachedDiskTempPath); err == nil {
			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 0 && temp < 100 {
					return temp
				}
			}
		}
		cachedDiskTempPath = ""
		logInfoModule("disk", "Disk temperature path changed, rescanning")
	}

	hwmonDirs, err := ioutil.ReadDir("/sys/class/hwmon")
	if err != nil {
		return 0.0
	}

	for _, hwmon := range hwmonDirs {
		hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", hwmon.Name())

		nameBytes, err := ioutil.ReadFile(hwmonPath + "/name")
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
			tempBytes, err := ioutil.ReadFile(tempPath)
			if err != nil {
				continue
			}

			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 0 && temp < 100 {
					cachedDiskTempPath = tempPath
					logInfoModule("disk", "Found disk temperature sensor: %s (%.1f°C)", hwmonName, temp)
					return temp
				}
			}
		}
	}
	return 0.0
}

// getRealCPUFrequency gets real CPU frequency (min and max) (Linux)
func getRealCPUFrequency() (float64, float64) {
	// Try to read max frequency from /sys
	maxFreqBytes, err := ioutil.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")
	if err == nil {
		maxFreqStr := strings.TrimSpace(string(maxFreqBytes))
		if maxFreqInt, err := strconv.Atoi(maxFreqStr); err == nil {
			maxFreq := float64(maxFreqInt) / 1000.0 // Convert from kHz to MHz

			// Try to read current frequency
			curFreqBytes, err := ioutil.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq")
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

func getRealGPUTemperature() float64 {
	initializeCache()
	if cachedGPUInfo == nil {
		return 0.0
	}

	if cachedGPUTempPath != "" {
		if tempBytes, err := ioutil.ReadFile(cachedGPUTempPath); err == nil {
			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 0 && temp < 120 {
					return temp
				}
			}
		}
		cachedGPUTempPath = ""
		logInfoModule("gpu", "GPU temperature path changed, rescanning")
	}

	hwmonFiles, err := ioutil.ReadDir("/sys/class/hwmon")
	if err != nil {
		return 0.0
	}

	for _, file := range hwmonFiles {
		hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", file.Name())

		nameBytes, err := ioutil.ReadFile(hwmonPath + "/name")
		if err != nil {
			continue
		}

		name := strings.TrimSpace(string(nameBytes))
		if strings.Contains(name, "nouveau") || strings.Contains(name, "amdgpu") {
			tempPath := hwmonPath + "/temp1_input"
			tempBytes, err := ioutil.ReadFile(tempPath)
			if err != nil {
				continue
			}

			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 0 && temp < 120 {
					cachedGPUTempPath = tempPath
					logInfoModule("gpu", "Found GPU temperature sensor: %s (%.1f°C)", name, temp)
					return temp
				}
			}
		}
	}

	return 0.0
}

// getRealGPUFrequency gets real GPU frequency (Linux)
func getRealGPUFrequency() float64 {
	initializeCache()
	if cachedGPUInfo == nil {
		return 0.0
	}

	// Try to read GPU frequency from /sys/class/drm
	gpuFiles, err := ioutil.ReadDir("/sys/class/drm")
	if err != nil {
		return 0.0
	}

	for _, file := range gpuFiles {
		if strings.HasPrefix(file.Name(), "card") && !strings.Contains(file.Name(), "-") {
			freqPath := fmt.Sprintf("/sys/class/drm/%s/device/pp_dpm_sclk", file.Name())
			freqBytes, err := ioutil.ReadFile(freqPath)
			if err != nil {
				continue
			}

			// Parse current frequency (format varies by driver)
			lines := strings.Split(string(freqBytes), "\n")
			for _, line := range lines {
				if strings.Contains(line, "*") { // Current frequency marked with *
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						freqStr := strings.TrimSuffix(parts[1], "Mhz")
						if freq, err := strconv.Atoi(freqStr); err == nil {
							return float64(freq)
						}
					}
				}
			}
		}
	}

	return 0.0
}

// getRealGPUUsage gets real GPU usage from /sys files (Linux)
func getRealGPUUsage() float64 {
	initializeCache()
	if cachedGPUInfo == nil {
		return 0.0
	}

	// Try to read GPU usage from /sys/class/drm
	gpuFiles, err := ioutil.ReadDir("/sys/class/drm")
	if err != nil {
		return 0.0
	}

	for _, file := range gpuFiles {
		if strings.HasPrefix(file.Name(), "card") && !strings.Contains(file.Name(), "-") {
			usagePath := fmt.Sprintf("/sys/class/drm/%s/device/gpu_busy_percent", file.Name())
			usageBytes, err := ioutil.ReadFile(usagePath)
			if err != nil {
				// Try alternative path
				usagePath = fmt.Sprintf("/sys/class/drm/%s/device/engine/render/load", file.Name())
				usageBytes, err = ioutil.ReadFile(usagePath)
				if err != nil {
					continue
				}
			}

			usageStr := strings.TrimSpace(string(usageBytes))
			if usage, err := strconv.Atoi(usageStr); err == nil {
				return float64(usage)
			}
		}
	}

	return 0.0
}

// getRealGPUFanSpeed gets real GPU fan speed (Linux)
func getRealGPUFanSpeed() int {
	initializeCache()
	if cachedGPUInfo == nil {
		return 0
	}

	// Try to read GPU fan speed from hwmon
	hwmonFiles, err := ioutil.ReadDir("/sys/class/hwmon")
	if err != nil {
		return 0
	}

	for _, file := range hwmonFiles {
		hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", file.Name())

		// Check if this is a GPU-related sensor
		nameBytes, err := ioutil.ReadFile(hwmonPath + "/name")
		if err != nil {
			continue
		}

		name := strings.TrimSpace(string(nameBytes))
		if strings.Contains(name, "nouveau") || strings.Contains(name, "amdgpu") {
			// Try to read fan speed
			fanPath := hwmonPath + "/fan1_input"
			fanBytes, err := ioutil.ReadFile(fanPath)
			if err != nil {
				continue
			}

			fanStr := strings.TrimSpace(string(fanBytes))
			if fanSpeed, err := strconv.Atoi(fanStr); err == nil {
				return fanSpeed
			}
		}
	}

	return 0
}

// getRealAllFans gets real system fan information from /sys/class/hwmon (Linux)
func getRealAllFans() []FanInfo {
	var fans []FanInfo

	hwmonFiles, err := ioutil.ReadDir("/sys/class/hwmon")
	if err != nil {
		return fans
	}

	for _, file := range hwmonFiles {
		hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", file.Name())

		// Read hwmon name
		nameBytes, err := ioutil.ReadFile(hwmonPath + "/name")
		if err != nil {
			continue
		}
		hwmonName := strings.TrimSpace(string(nameBytes))

		// Look for fan inputs
		hwmonContents, err := ioutil.ReadDir(hwmonPath)
		if err != nil {
			continue
		}

		for _, hwmonFile := range hwmonContents {
			if strings.HasPrefix(hwmonFile.Name(), "fan") && strings.HasSuffix(hwmonFile.Name(), "_input") {
				fanPath := hwmonPath + "/" + hwmonFile.Name()
				fanBytes, err := ioutil.ReadFile(fanPath)
				if err != nil {
					continue
				}

				fanStr := strings.TrimSpace(string(fanBytes))
				if fanSpeed, err := strconv.Atoi(fanStr); err == nil && fanSpeed > 0 {
					// Try to read fan label
					labelPath := hwmonPath + "/" + strings.Replace(hwmonFile.Name(), "_input", "_label", 1)
					labelBytes, err := ioutil.ReadFile(labelPath)
					var fanName string
					if err == nil {
						fanName = strings.TrimSpace(string(labelBytes))
					} else {
						// Generate name from hwmon name and fan number
						fanNum := strings.TrimPrefix(hwmonFile.Name(), "fan")
						fanNum = strings.TrimSuffix(fanNum, "_input")
						fanName = fmt.Sprintf("%s Fan %s", hwmonName, fanNum)
					}

					fans = append(fans, FanInfo{Name: fanName, Speed: fanSpeed})
				}
			}
		}
	}

	return fans
}

type NetworkInfoData struct {
	IP            string
	UploadSpeed   float64
	DownloadSpeed float64
}

var (
	lastNetTime  time.Time
	lastNetStats map[string]uint64
)

func getNetworkInfo() NetworkInfoData {
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
	if !lastNetTime.IsZero() {
		duration := now.Sub(lastNetTime).Seconds()
		if duration > 0 {
			statsPath := "/proc/net/dev"
			data, err := ioutil.ReadFile(statsPath)
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

func getGPUFPS() float64 {
	// GPU FPS monitoring is complex and requires game-specific hooks
	// For now, we can try to estimate based on GPU usage patterns
	// This is not real FPS but gives an indication of GPU activity

	usage := getRealGPUUsage()
	if usage > 80 {
		// High usage might indicate gaming/rendering
		return 60.0 // Estimate
	} else if usage > 50 {
		return 30.0 // Moderate activity
	}

	// Real FPS monitoring would require:
	// - OpenGL/Vulkan/DirectX hooks
	// - Game-specific APIs
	// - Frame buffer analysis
	logDebugModule("gpu", "GPU FPS monitoring not fully implemented, returning usage-based estimate")
	return 0.0
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
	if data, err := ioutil.ReadFile("/proc/cpuinfo"); err == nil {
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
	if entries, err := ioutil.ReadDir("/sys/devices/system/cpu/"); err == nil {
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
	if data, err := ioutil.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"); err == nil {
		if freq, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			cpuInfo.MaxFreq = freq / 1000 // Convert from kHz to MHz
		}
	}

	if data, err := ioutil.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_min_freq"); err == nil {
		if freq, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			cpuInfo.MinFreq = freq / 1000 // Convert from kHz to MHz
		}
	}

	return cpuInfo
}

// detectLinuxGPUInfo detects detailed GPU information on Linux
func detectLinuxGPUInfo() *GPUInfo {
	gpuInfo := &GPUInfo{
		Model:       "Unknown GPU",
		Vendor:      "unknown",
		Memory:      0,
		MemoryUsed:  0,
		FanCount:    0,
		Fans:        []FanInfo{},
		Temperature: 0,
		Usage:       0,
		Frequency:   0,
	}

	// Priority: Discrete GPUs only (NVIDIA > AMD with VRAM)
	var discreteGPUs []*GPUInfo

	// Check for NVIDIA GPU (discrete)
	if _, err := ioutil.ReadFile("/proc/driver/nvidia/version"); err == nil {
		nvidiaGPU := detectNVIDIAGPU()
		if nvidiaGPU != nil {
			discreteGPUs = append(discreteGPUs, nvidiaGPU)
		}
	}

	// Check for AMD discrete GPU (with dedicated VRAM)
	amdGPUs := detectAMDDiscreteGPUs()
	discreteGPUs = append(discreteGPUs, amdGPUs...)

	// Select the best discrete GPU
	if len(discreteGPUs) > 0 {
		// Priority: NVIDIA > AMD with most VRAM
		var bestGPU *GPUInfo

		// First, check for NVIDIA GPUs
		for _, candidate := range discreteGPUs {
			if candidate.Vendor == "NVIDIA" {
				if bestGPU == nil || candidate.Memory > bestGPU.Memory {
					bestGPU = candidate
				}
			}
		}

		// If no NVIDIA found, select AMD GPU with most VRAM
		if bestGPU == nil {
			for _, candidate := range discreteGPUs {
				if candidate.Vendor == "AMD" {
					if bestGPU == nil || candidate.Memory > bestGPU.Memory {
						bestGPU = candidate
					}
				}
			}
		}

		if bestGPU != nil {
			*gpuInfo = *bestGPU
			logInfoModule("gpu", "Selected discrete GPU: %s (%d MB VRAM)", bestGPU.Model, bestGPU.Memory)
			return gpuInfo
		}
	}

	logWarnModule("gpu", "No discrete GPU found, GPU monitoring unavailable")
	return gpuInfo
}

// detectNVIDIAGPU detects NVIDIA GPU information
func detectNVIDIAGPU() *GPUInfo {
	nvidiaGPU := &GPUInfo{
		Model:      "NVIDIA GPU",
		Vendor:     "NVIDIA",
		Memory:     0,
		MemoryUsed: 0,
		FanCount:   0,
		Fans:       []FanInfo{},
	}

	// Try to get GPU model from nvidia driver files
	if entries, err := ioutil.ReadDir("/proc/driver/nvidia/gpus/"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				gpuPath := fmt.Sprintf("/proc/driver/nvidia/gpus/%s", entry.Name())
				if infoData, err := ioutil.ReadFile(gpuPath + "/information"); err == nil {
					lines := strings.Split(string(infoData), "\n")
					for _, line := range lines {
						if strings.Contains(line, "Model:") {
							parts := strings.SplitN(line, ":", 2)
							if len(parts) == 2 {
								model := strings.TrimSpace(parts[1])
								model = strings.ReplaceAll(model, "NVIDIA ", "")
								model = strings.ReplaceAll(model, "GeForce ", "")
								nvidiaGPU.Model = "NVIDIA " + model
							}
						}
					}
				}
				break // Use first GPU
			}
		}
	}

	// Try to get memory information from DRM
	if entries, err := ioutil.ReadDir("/sys/class/drm/"); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "card") && !strings.Contains(entry.Name(), "-") {
				devicePath := fmt.Sprintf("/sys/class/drm/%s/device", entry.Name())
				if vendorData, err := ioutil.ReadFile(devicePath + "/vendor"); err == nil {
					vendorID := strings.TrimSpace(string(vendorData))
					if vendorID == "0x10de" { // NVIDIA vendor ID
						// Try to get memory info
						if memData, err := ioutil.ReadFile(devicePath + "/mem_info_vram_total"); err == nil {
							if memBytes, err := strconv.ParseInt(strings.TrimSpace(string(memData)), 10, 64); err == nil {
								nvidiaGPU.Memory = memBytes / (1024 * 1024) // Convert to MB
							}
						}
						break
					}
				}
			}
		}
	}

	return nvidiaGPU
}

// detectAMDDiscreteGPUs detects AMD discrete GPUs (with dedicated VRAM)
func detectAMDDiscreteGPUs() []*GPUInfo {
	var amdGPUs []*GPUInfo

	if entries, err := ioutil.ReadDir("/sys/class/drm/"); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "card") && !strings.Contains(entry.Name(), "-") {
				devicePath := fmt.Sprintf("/sys/class/drm/%s/device", entry.Name())

				// Check vendor
				if vendorData, err := ioutil.ReadFile(devicePath + "/vendor"); err == nil {
					vendorID := strings.TrimSpace(string(vendorData))
					if vendorID == "0x1002" { // AMD vendor ID
						// Check if it has dedicated VRAM (discrete GPU)
						if memData, err := ioutil.ReadFile(devicePath + "/mem_info_vram_total"); err == nil {
							if memBytes, err := strconv.ParseInt(strings.TrimSpace(string(memData)), 10, 64); err == nil && memBytes > 0 {
								amdGPU := &GPUInfo{
									Vendor:     "AMD",
									Memory:     memBytes / (1024 * 1024), // Convert to MB
									MemoryUsed: 0,
									FanCount:   0,
									Fans:       []FanInfo{},
								}

								// Try to get model name
								modelFound := false
								if nameData, err := ioutil.ReadFile(devicePath + "/device_name"); err == nil {
									name := strings.TrimSpace(string(nameData))
									if name != "" {
										amdGPU.Model = "AMD " + name
										modelFound = true
									}
								}

								if !modelFound {
									if deviceData, err := ioutil.ReadFile(devicePath + "/device"); err == nil {
										deviceID := strings.TrimSpace(string(deviceData))

										// Map known AMD device IDs to proper names
										switch deviceID {
										case "0x731f":
											amdGPU.Model = "AMD Radeon RX 5700 XT"
										case "0x7340":
											amdGPU.Model = "AMD Radeon RX 5700"
										case "0x13c0":
											amdGPU.Model = "AMD Radeon Graphics (Integrated)"
										case "0x1638":
											amdGPU.Model = "AMD Radeon RX 5600 XT"
										case "0x67df":
											amdGPU.Model = "AMD Radeon RX 480"
										case "0x67ef":
											amdGPU.Model = "AMD Radeon RX 460"
										default:
											amdGPU.Model = "AMD GPU (" + deviceID + ")"
										}
									} else {
										amdGPU.Model = "AMD Discrete GPU"
									}
								}

								// Try to get used memory
								if memUsedData, err := ioutil.ReadFile(devicePath + "/mem_info_vram_used"); err == nil {
									if memUsedBytes, err := strconv.ParseInt(strings.TrimSpace(string(memUsedData)), 10, 64); err == nil {
										amdGPU.MemoryUsed = memUsedBytes / (1024 * 1024) // Convert to MB
									}
								}

								logDebugModule("gpu", "Found AMD GPU: %s with %d MB VRAM", amdGPU.Model, amdGPU.Memory)
								amdGPUs = append(amdGPUs, amdGPU)
							}
						}
					}
				}
			}
		}
	}

	return amdGPUs
}

// detectLinuxDiskInfo detects detailed disk information on Linux
func detectLinuxDiskInfo() []*DiskInfo {
	var disks []*DiskInfo

	// Read block devices from /sys/block
	if entries, err := ioutil.ReadDir("/sys/block"); err == nil {
		for _, entry := range entries {
			// Skip virtual devices, loop devices, ram disks, etc.
			if strings.HasPrefix(entry.Name(), "loop") ||
				strings.HasPrefix(entry.Name(), "ram") ||
				strings.HasPrefix(entry.Name(), "dm-") ||
				strings.HasPrefix(entry.Name(), "zram") ||
				strings.HasPrefix(entry.Name(), "md") ||
				strings.HasPrefix(entry.Name(), "sr") ||
				strings.HasPrefix(entry.Name(), "fd") {
				continue
			}

			// Only include physical storage devices (nvme, sd, hd)
			if !strings.HasPrefix(entry.Name(), "nvme") &&
				!strings.HasPrefix(entry.Name(), "sd") &&
				!strings.HasPrefix(entry.Name(), "hd") {
				continue
			}

			diskPath := fmt.Sprintf("/sys/block/%s", entry.Name())
			disk := &DiskInfo{
				Name:        entry.Name(),
				Model:       "Unknown",
				Size:        0,
				Temperature: 0,
				ReadSpeed:   0,
				WriteSpeed:  0,
				Usage:       0,
			}

			// Get disk model
			if modelData, err := ioutil.ReadFile(diskPath + "/device/model"); err == nil {
				disk.Model = strings.TrimSpace(string(modelData))
			}

			// Get disk size (in sectors, multiply by 512 to get bytes)
			if sizeData, err := ioutil.ReadFile(diskPath + "/size"); err == nil {
				if sectors, err := strconv.ParseInt(strings.TrimSpace(string(sizeData)), 10, 64); err == nil {
					disk.Size = (sectors * 512) / (1024 * 1024 * 1024) // Convert to GB
				}
			}

			// Try to get disk temperature from hwmon
			disk.Temperature = getDiskTemperatureByName(entry.Name())

			// Get disk I/O stats and calculate real-time speeds
			if statData, err := ioutil.ReadFile(diskPath + "/stat"); err == nil {
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
					if lastStats, exists := lastDiskStats[entry.Name()]; exists {
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
					lastDiskStats[entry.Name()] = &DiskIOSnapshot{
						ReadSectors:  currentReadSectors,
						WriteSectors: currentWriteSectors,
						Timestamp:    now,
					}
				}
			}

			// Calculate disk usage percentage using statvfs
			disk.Usage = getDiskUsagePercentage(entry.Name())

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
		if tempData, err := ioutil.ReadFile(p); err == nil {
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
	if hwmonEntries, err := ioutil.ReadDir("/sys/class/hwmon"); err == nil {
		for _, entry := range hwmonEntries {
			hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", entry.Name())

			// Check if this hwmon is for our disk
			if nameData, err := ioutil.ReadFile(hwmonPath + "/name"); err == nil {
				name := strings.TrimSpace(string(nameData))

				// Check for disk-specific hwmon names
				if strings.Contains(strings.ToLower(name), strings.ToLower(deviceName)) ||
					strings.Contains(strings.ToLower(name), "drivetemp") ||
					(strings.Contains(strings.ToLower(name), "nvme") && strings.Contains(deviceName, "nvme")) ||
					(strings.Contains(strings.ToLower(name), "ata") && strings.HasPrefix(deviceName, "sd")) {

					// Look for temperature sensors
					if tempEntries, err := ioutil.ReadDir(hwmonPath); err == nil {
						for _, tempEntry := range tempEntries {
							if strings.HasPrefix(tempEntry.Name(), "temp") && strings.HasSuffix(tempEntry.Name(), "_input") {
								tempPath := fmt.Sprintf("%s/%s", hwmonPath, tempEntry.Name())
								if tempData, err := ioutil.ReadFile(tempPath); err == nil {
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
	if hwmonEntries, err := ioutil.ReadDir(blockPath); err == nil {
		for _, entry := range hwmonEntries {
			hwmonPath := fmt.Sprintf("%s/%s", blockPath, entry.Name())
			if tempEntries, err := ioutil.ReadDir(hwmonPath); err == nil {
				for _, tempEntry := range tempEntries {
					if strings.HasPrefix(tempEntry.Name(), "temp") && strings.HasSuffix(tempEntry.Name(), "_input") {
						tempPath := fmt.Sprintf("%s/%s", hwmonPath, tempEntry.Name())
						if tempData, err := ioutil.ReadFile(tempPath); err == nil {
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
		nvmePath := fmt.Sprintf("/sys/class/nvme/%s", strings.Split(deviceName, "n")[0])
		if nvmeEntries, err := ioutil.ReadDir(nvmePath); err == nil {
			for _, entry := range nvmeEntries {
				if strings.HasPrefix(entry.Name(), "hwmon") {
					hwmonPath := fmt.Sprintf("%s/%s", nvmePath, entry.Name())
					if tempEntries, err := ioutil.ReadDir(hwmonPath); err == nil {
						for _, tempEntry := range tempEntries {
							if strings.HasPrefix(tempEntry.Name(), "temp") && strings.HasSuffix(tempEntry.Name(), "_input") {
								tempPath := fmt.Sprintf("%s/%s", hwmonPath, tempEntry.Name())
								if tempData, err := ioutil.ReadFile(tempPath); err == nil {
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

// getDiskUsagePercentage calculates disk usage percentage for a device
func getDiskUsagePercentage(deviceName string) float64 {
	// Try to find the mount point for this device
	mountsData, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return 0.0
	}

	lines := strings.Split(string(mountsData), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			device := fields[0]
			mountPoint := fields[1]

			// Check if this mount point corresponds to our device
			if strings.Contains(device, deviceName) ||
				(strings.HasPrefix(device, "/dev/") && strings.Contains(device, deviceName)) {

				// Use statvfs to get filesystem statistics
				if usage := getFilesystemUsage(mountPoint); usage > 0 {
					return usage
				}
			}
		}
	}

	// If no specific mount found, try common mount points
	commonMounts := []string{"/", "/home", "/var", "/tmp"}
	for _, mount := range commonMounts {
		if usage := getFilesystemUsage(mount); usage > 0 {
			return usage
		}
	}

	return 0.0
}

// getFilesystemUsage gets filesystem usage percentage for a mount point
func getFilesystemUsage(mountPoint string) float64 {
	// This would use syscall.Statfs on Linux
	// For now, return a placeholder that tries to read from df command
	return 0.0
}
