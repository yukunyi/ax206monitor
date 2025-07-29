//go:build linux

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	cachedCPUTempPath  string
	cachedDiskTempPath string
	cachedGPUTempPath  string
)

func getRealCPUTemperature() float64 {
	if cachedCPUTempPath != "" {
		if tempBytes, err := ioutil.ReadFile(cachedCPUTempPath); err == nil {
			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > 20 && temp < 150 {
					return temp
				}
			}
		}
		cachedCPUTempPath = ""
		logInfoModule("cpu", "CPU temperature path changed, rescanning")
	}

	maxTemp := 0.0
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

		isCPUSensor := false
		if hwmonName == "k10temp" || hwmonName == "coretemp" || hwmonName == "zenpower" ||
			strings.Contains(hwmonName, "cpu") || strings.Contains(hwmonName, "package") {
			isCPUSensor = true
		}

		if isCPUSensor {
			tempPath := hwmonPath + "/temp1_input"
			tempBytes, err := ioutil.ReadFile(tempPath)
			if err != nil {
				continue
			}

			tempStr := strings.TrimSpace(string(tempBytes))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				temp := float64(tempInt) / 1000.0
				if temp > maxTemp && temp < 150 && temp > 20 {
					maxTemp = temp
					cachedCPUTempPath = tempPath
					logInfoModule("cpu", "Found CPU temperature sensor: %s (%.1f°C)", hwmonName, temp)
				}
			}
		} else {
			tempFiles, err := ioutil.ReadDir(hwmonPath)
			if err != nil {
				continue
			}

			for _, tempFile := range tempFiles {
				if strings.HasPrefix(tempFile.Name(), "temp") && strings.HasSuffix(tempFile.Name(), "_input") {
					tempNum := strings.TrimSuffix(strings.TrimPrefix(tempFile.Name(), "temp"), "_input")
					labelPath := fmt.Sprintf("%s/temp%s_label", hwmonPath, tempNum)
					inputPath := fmt.Sprintf("%s/temp%s_input", hwmonPath, tempNum)

					labelBytes, err := ioutil.ReadFile(labelPath)
					if err != nil {
						continue
					}

					label := strings.TrimSpace(string(labelBytes))
					if strings.Contains(strings.ToLower(label), "cpu") ||
						strings.Contains(strings.ToLower(label), "package") ||
						strings.Contains(strings.ToLower(label), "core") {

						tempBytes, err := ioutil.ReadFile(inputPath)
						if err != nil {
							continue
						}

						tempStr := strings.TrimSpace(string(tempBytes))
						if tempInt, err := strconv.Atoi(tempStr); err == nil {
							temp := float64(tempInt) / 1000.0
							if temp > maxTemp && temp < 150 && temp > 20 {
								maxTemp = temp
								cachedCPUTempPath = inputPath
								logInfoModule("cpu", "Found CPU temperature sensor: %s/%s (%.1f°C)", hwmonName, label, temp)
							}
						}
					}
				}
			}
		}
	}
	return maxTemp
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
	if cachedGPUModel == "" {
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
	if cachedGPUModel == "" {
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
	if cachedGPUModel == "" {
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
	if cachedGPUModel == "" {
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
	return 0.0
}
