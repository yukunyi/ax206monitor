package main

import (
	"io/ioutil"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	cachedCPUInfo    *CPUInfo
	cachedGPUInfo    *GPUInfo
	cachedDiskInfo   []*DiskInfo
	cacheInitMutex   sync.Once
	diskInfoMutex    sync.RWMutex
	lastDiskUpdate   time.Time
	diskUpdatePeriod = 1 * time.Second

	// 无锁读取用原子存储
	diskInfoStore atomic.Value // []*DiskInfo

	defaultDiskMutex    sync.Mutex
	lastDefaultDiskName string

	// 渲染门控
	renderAccessMutex   sync.Mutex
	lastRenderAccess    time.Time
	renderAccessTimeout = 5 * time.Second

	diskSamplerOnce sync.Once

	// 根设备缓存
	rootDeviceMutex     sync.Mutex
	cachedRootDevice    string
	lastRootDeviceCheck time.Time
	rootDeviceCacheTTL  = 5 * time.Second
	rootDeviceFetchRun  bool
)

func noteRenderAccess() {
	renderAccessMutex.Lock()
	lastRenderAccess = time.Now()
	renderAccessMutex.Unlock()
}

func isRenderActive() bool {
	renderAccessMutex.Lock()
	defer renderAccessMutex.Unlock()
	return time.Since(lastRenderAccess) <= renderAccessTimeout
}

func initializeCache() {
	cacheInitMutex.Do(func() {
		cachedCPUInfo = detectCPUInfo()
		cachedGPUInfo = detectGPUInfo()
		go func() {
			updateDiskInfo()
			startDiskSampler()
			printSystemInfo()
		}()
	})
}

// updateDiskInfo updates disk information if enough time has passed
func updateDiskInfo() {
	now := time.Now()
	diskInfoMutex.Lock()
	if now.Sub(lastDiskUpdate) < diskUpdatePeriod {
		diskInfoMutex.Unlock()
		return
	}
	diskInfoMutex.Unlock()

	// 计算新数据不持锁
	newDisks := detectDiskInfo()
	if len(newDisks) > 1 {
		sort.Slice(newDisks, func(i, j int) bool { return newDisks[i].Name < newDisks[j].Name })
	}

	// 写入缓存与时间戳
	diskInfoMutex.Lock()
	cachedDiskInfo = newDisks
	lastDiskUpdate = now
	diskInfoMutex.Unlock()
	diskInfoStore.Store(newDisks)
}

// getCachedDiskInfo returns current disk information without lock (atomic)
func getCachedDiskInfo() []*DiskInfo {
	initializeCache()
	if v := diskInfoStore.Load(); v != nil {
		if disks, ok := v.([]*DiskInfo); ok {
			return disks
		}
	}
	diskInfoMutex.RLock()
	defer diskInfoMutex.RUnlock()
	return cachedDiskInfo
}

func startDiskSampler() {
	diskSamplerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(diskUpdatePeriod)
			defer ticker.Stop()
			for range ticker.C {
				if !isRenderActive() {
					continue
				}
				updateDiskInfo()
				refreshRootDeviceIfNeeded()
			}
		}()
	})
}

func refreshRootDeviceIfNeeded() {
	rootDeviceMutex.Lock()
	now := time.Now()
	if now.Sub(lastRootDeviceCheck) < rootDeviceCacheTTL || rootDeviceFetchRun {
		rootDeviceMutex.Unlock()
		return
	}
	rootDeviceFetchRun = true
	rootDeviceMutex.Unlock()
	go func() {
		dev := detectRootDevice()
		rootDeviceMutex.Lock()
		cachedRootDevice = dev
		lastRootDeviceCheck = time.Now()
		rootDeviceFetchRun = false
		rootDeviceMutex.Unlock()
	}()
}

func getCachedRootDevice() string {
	rootDeviceMutex.Lock()
	now := time.Now()
	dev := cachedRootDevice
	expired := now.Sub(lastRootDeviceCheck) >= rootDeviceCacheTTL
	already := rootDeviceFetchRun
	rootDeviceMutex.Unlock()
	if expired && !already {
		refreshRootDeviceIfNeeded()
	}
	return dev
}

func getDefaultDiskIndex() int {
	initializeCache()
	disks := getCachedDiskInfo()
	if len(disks) == 0 {
		return -1
	}
	device := getCachedRootDevice()
	var chosenName string
	var idx int
	if device == "" {
		chosenName = disks[0].Name
		idx = 0
	} else {
		base := baseDeviceName(device)
		chosenName = base
		found := false
		for i, d := range disks {
			if d.Name == base {
				idx = i
				found = true
				break
			}
		}
		if !found {
			chosenName = disks[0].Name
			idx = 0
		}
	}
	defaultDiskMutex.Lock()
	if chosenName != "" && chosenName != lastDefaultDiskName {
		lastDefaultDiskName = chosenName
		logInfoModule("disk", "Default disk: %s", chosenName)
	}
	defaultDiskMutex.Unlock()
	return idx
}

func detectRootDevice() string {
	if data, err := ioutil.ReadFile("/proc/mounts"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "/" {
				return fields[0]
			}
		}
	}
	return ""
}

func baseDeviceName(device string) string {
	dev := device
	if strings.HasPrefix(dev, "/dev/") {
		dev = dev[5:]
	}
	if strings.HasPrefix(dev, "nvme") {
		if idx := strings.Index(dev, "p"); idx > 0 {
			return dev[:idx]
		}
		return dev
	}
	if strings.HasPrefix(dev, "mmcblk") {
		if idx := strings.Index(dev, "p"); idx > 0 {
			return dev[:idx]
		}
		return dev
	}
	if len(dev) >= 3 && (strings.HasPrefix(dev, "sd") || strings.HasPrefix(dev, "hd")) {
		i := len(dev) - 1
		for i >= 0 && dev[i] >= '0' && dev[i] <= '9' {
			i--
		}
		return dev[:i+1]
	}
	return dev
}

func printSystemInfo() {
	logInfo("=== System Information ===")
	if cachedCPUInfo != nil {
		logInfo("CPU: %s", cachedCPUInfo.Model)
		logInfo("CPU Vendor: %s", cachedCPUInfo.Vendor)
		logInfo("CPU Architecture: %s", cachedCPUInfo.Architecture)
		logInfo("CPU Cores: %d, Threads: %d", cachedCPUInfo.Cores, cachedCPUInfo.Threads)
		logInfo("CPU Frequency: %.0f MHz - %.0f MHz", cachedCPUInfo.MinFreq, cachedCPUInfo.MaxFreq)
	}
	if cachedGPUInfo != nil {
		logInfo("GPU: %s (%s)", cachedGPUInfo.Model, cachedGPUInfo.Vendor)
		if cachedGPUInfo.Memory > 0 {
			logInfo("GPU Memory: %d MB", cachedGPUInfo.Memory)
		}
		if cachedGPUInfo.FanCount > 0 {
			logInfo("GPU Fans: %d", cachedGPUInfo.FanCount)
		}
	}
	disks := getCachedDiskInfo()
	if len(disks) > 0 {
		logInfo("Disks: %d detected", len(disks))
		for i, disk := range disks {
			if i < 3 {
				logInfo("Disk %d: %s (%s) - %.0f GB", i+1, disk.Name, disk.Model, float64(disk.Size))
			}
		}
	}
	logInfo("OS: %s %s", runtime.GOOS, runtime.GOARCH)
	logInfo("========================")
}
