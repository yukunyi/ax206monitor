package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type FanMonitor struct {
	*BaseMonitorItem
	fanIndex int
}

func NewFanMonitor(fanIndex int, fanName string) *FanMonitor {
	name := fmt.Sprintf("fan%d", fanIndex)
	label := fanName
	if label == "" {
		label = fmt.Sprintf("Fan %d", fanIndex)
	}

	return &FanMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 5000, "RPM", 0),
		fanIndex:        fanIndex,
	}
}

func (f *FanMonitor) Update() error {
	fans := GetAvailableFans()
	// fanIndex is 1-based, array is 0-based
	if f.fanIndex > 0 && f.fanIndex <= len(fans) {
		f.SetValue(fans[f.fanIndex-1].Speed)
		f.SetAvailable(true)
	} else {
		f.SetAvailable(false)
	}
	return nil
}

func GetAvailableFans() []FanInfo {
	if runtime.GOOS == "windows" {
		return getWindowsFanInfo()
	}
	return getLinuxFanInfo()
}

func getWindowsFanInfo() []FanInfo {
	// Use cached GPU info for GPU fans
	fans := []FanInfo{}

	if cachedGPUInfo != nil && len(cachedGPUInfo.Fans) > 0 {
		fans = append(fans, cachedGPUInfo.Fans...)
	}

	// Add system fans (placeholder for Windows implementation)
	// In a real implementation, this would use WMI or hardware monitoring libraries
	systemFans := []FanInfo{
		{Name: "CPU Fan", Speed: 1200, Index: 1},
		{Name: "Case Fan 1", Speed: 800, Index: 2},
		{Name: "Case Fan 2", Speed: 850, Index: 3},
	}

	fans = append(fans, systemFans...)
	return fans
}

func getLinuxFanInfo() []FanInfo {
	if runtime.GOOS != "linux" {
		return []FanInfo{}
	}

	fans := []FanInfo{}
	cpuFanFound := false
	gpuFanFound := false
	sysFanCount := 0

	hwmonDirs := []string{"/sys/class/hwmon"}
	for _, hwmonDir := range hwmonDirs {
		if entries, err := ioutil.ReadDir(hwmonDir); err == nil {
			logDebugModule("fan", "Found %d hwmon entries", len(entries))
			for _, entry := range entries {
				// hwmon entries are usually symlinks, so we need to check if they point to directories
				hwmonPath := filepath.Join(hwmonDir, entry.Name())
				if stat, err := os.Stat(hwmonPath); err != nil || !stat.IsDir() {
					logDebugModule("fan", "Skipping non-directory or inaccessible: %s", entry.Name())
					continue
				}

				logDebugModule("fan", "Checking hwmon path: %s", hwmonPath)

				// Read hwmon name to identify the device
				nameFile := filepath.Join(hwmonPath, "name")
				var deviceName string
				if nameData, err := ioutil.ReadFile(nameFile); err == nil {
					deviceName = strings.TrimSpace(string(nameData))
					logDebugModule("fan", "Device name: %s", deviceName)
				} else {
					// Skip if we can't read the name
					logDebugModule("fan", "Cannot read name file %s: %v", nameFile, err)
					continue
				}

				// Find all fan input files
				fanFiles, err := filepath.Glob(filepath.Join(hwmonPath, "fan*_input"))
				if err != nil {
					logDebugModule("fan", "Error globbing fan files in %s: %v", hwmonPath, err)
					continue
				}

				logDebugModule("fan", "Found %d fan files in %s", len(fanFiles), hwmonPath)
				for _, fanFile := range fanFiles {
					logDebugModule("fan", "Processing fan file: %s", fanFile)
					if data, err := ioutil.ReadFile(fanFile); err == nil {
						speedStr := strings.TrimSpace(string(data))
						if speed, err := strconv.Atoi(speedStr); err == nil && speed > 0 {
							// Determine fan type and name based on device name and label
							var fanName string

							// Try to get a more descriptive name from label file first
							labelFile := strings.Replace(fanFile, "_input", "_label", 1)
							labelName := ""
							if labelData, err := ioutil.ReadFile(labelFile); err == nil {
								labelName = strings.TrimSpace(string(labelData))
							}

							// Categorize fan based on device name and label
							deviceLower := strings.ToLower(deviceName)
							labelLower := strings.ToLower(labelName)

							if strings.Contains(deviceLower, "cpu") ||
								strings.Contains(deviceLower, "coretemp") ||
								strings.Contains(deviceLower, "k10temp") ||
								strings.Contains(deviceLower, "zenpower") ||
								strings.Contains(labelLower, "cpu") {
								if !cpuFanFound {
									fanName = "CPU Fan"
									cpuFanFound = true
								} else {
									continue // Skip additional CPU fans
								}
							} else if strings.Contains(deviceLower, "gpu") ||
								strings.Contains(deviceLower, "nouveau") ||
								strings.Contains(deviceLower, "amdgpu") ||
								strings.Contains(deviceLower, "radeon") ||
								strings.Contains(labelLower, "gpu") {
								if !gpuFanFound {
									fanName = "GPU Fan"
									gpuFanFound = true
								} else {
									continue // Skip additional GPU fans
								}
							} else {
								// System fan - start from 1
								sysFanCount++
								if sysFanCount <= 10 { // Limit to sysfan1-10
									fanName = fmt.Sprintf("SysFan%d", sysFanCount)
								} else {
									continue // Skip fans beyond sysfan10
								}
							}

							fans = append(fans, FanInfo{
								Name:  fanName,
								Speed: speed,
								Index: len(fans) + 1,
							})

							// Debug logging
							logDebugModule("fan", "Found fan: %s = %d RPM (device: %s, label: %s)",
								fanName, speed, deviceName, labelName)
						}
					}
				}
			}
		}
	}

	logInfoModule("fan", "Detected %d fans total", len(fans))
	return fans
}

// FanSpeedMonitor displays fan speed for a specific fan type
type FanSpeedMonitor struct {
	*BaseMonitorItem
	fanType  string // "cpu", "gpu", or "system"
	fanIndex int    // For system fans (1-10)
}

func NewCPUFanMonitor() *FanSpeedMonitor {
	return &FanSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"cpu_fan_speed",
			"CPU Fan",
			0, 0,
			"RPM",
			0,
		),
		fanType:  "cpu",
		fanIndex: 0,
	}
}

func NewGPUFanMonitor() *FanSpeedMonitor {
	return &FanSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_fan_speed",
			"GPU Fan",
			0, 0,
			"RPM",
			0,
		),
		fanType:  "gpu",
		fanIndex: 0,
	}
}

func NewSystemFanMonitor(index int) *FanSpeedMonitor {
	return &FanSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			fmt.Sprintf("sysfan%d_speed", index),
			fmt.Sprintf("SysFan%d", index),
			0, 0,
			"RPM",
			0,
		),
		fanType:  "system",
		fanIndex: index,
	}
}

func (f *FanSpeedMonitor) Update() error {
	fans := GetAvailableFans()

	for _, fan := range fans {
		fanNameLower := strings.ToLower(fan.Name)

		if f.fanType == "cpu" && strings.Contains(fanNameLower, "cpu") {
			f.SetValue(fan.Speed)
			f.SetAvailable(true)
			return nil
		} else if f.fanType == "gpu" && strings.Contains(fanNameLower, "gpu") {
			f.SetValue(fan.Speed)
			f.SetAvailable(true)
			return nil
		} else if f.fanType == "system" && strings.Contains(fanNameLower, fmt.Sprintf("sysfan%d", f.fanIndex)) {
			f.SetValue(fan.Speed)
			f.SetAvailable(true)
			return nil
		}
	}

	f.SetAvailable(false)
	return nil
}
