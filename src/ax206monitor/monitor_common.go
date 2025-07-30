package main

import (
	"runtime"
	"sync"
)

// FanInfo represents fan information
type FanInfo struct {
	Name  string
	Speed int
	Index int
}

// CPUInfo represents detailed CPU information
type CPUInfo struct {
	Model        string
	Cores        int
	Threads      int
	Architecture string
	MaxFreq      float64
	MinFreq      float64
	Vendor       string
}

// GPUInfo represents detailed GPU information
type GPUInfo struct {
	Model       string
	Vendor      string
	Memory      int64 // Memory in MB
	MemoryUsed  int64 // Used memory in MB
	FanCount    int
	Fans        []FanInfo
	Temperature float64
	Usage       float64
	Frequency   float64
}

// DiskInfo represents detailed disk information
type DiskInfo struct {
	Name        string
	Model       string
	Size        int64 // Size in GB
	Temperature float64
	ReadSpeed   float64 // MB/s
	WriteSpeed  float64 // MB/s
	Usage       float64 // Usage percentage
}

var (
	cachedCPUInfo  *CPUInfo
	cachedGPUInfo  *GPUInfo
	cachedDiskInfo []*DiskInfo
	cacheInitMutex sync.Once
)

func initializeCache() {
	cacheInitMutex.Do(func() {
		cachedCPUInfo = detectCPUInfo()
		cachedGPUInfo = detectGPUInfo()
		cachedDiskInfo = detectDiskInfo()

		// Print detailed system information
		printSystemInfo()
	})
}

func printSystemInfo() {
	logInfo("=== System Information ===")

	// CPU Information
	if cachedCPUInfo != nil {
		logInfo("CPU: %s", cachedCPUInfo.Model)
		logInfo("CPU Vendor: %s", cachedCPUInfo.Vendor)
		logInfo("CPU Architecture: %s", cachedCPUInfo.Architecture)
		logInfo("CPU Cores: %d, Threads: %d", cachedCPUInfo.Cores, cachedCPUInfo.Threads)
		logInfo("CPU Frequency: %.0f MHz - %.0f MHz", cachedCPUInfo.MinFreq, cachedCPUInfo.MaxFreq)
	}

	// GPU Information
	if cachedGPUInfo != nil {
		logInfo("GPU: %s (%s)", cachedGPUInfo.Model, cachedGPUInfo.Vendor)
		if cachedGPUInfo.Memory > 0 {
			logInfo("GPU Memory: %d MB", cachedGPUInfo.Memory)
		}
		if cachedGPUInfo.FanCount > 0 {
			logInfo("GPU Fans: %d", cachedGPUInfo.FanCount)
		}
	}

	// Disk Information
	if len(cachedDiskInfo) > 0 {
		logInfo("Disks: %d detected", len(cachedDiskInfo))
		for i, disk := range cachedDiskInfo {
			if i < 3 { // Show first 3 disks
				logInfo("Disk %d: %s (%s) - %.0f GB", i+1, disk.Name, disk.Model, float64(disk.Size))
			}
		}
	}

	logInfo("OS: %s %s", runtime.GOOS, runtime.GOARCH)
	logInfo("========================")
}

func detectGPUModel() string {
	if cachedGPUInfo != nil {
		return cachedGPUInfo.Model
	}
	return "Generic GPU"
}
