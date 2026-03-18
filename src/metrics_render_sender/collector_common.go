package main

import (
	"runtime"
	"sort"
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

// DiskInfo represents detailed disk information
type DiskInfo struct {
	Name       string
	Model      string
	Size       int64   // Size in GB
	ReadSpeed  float64 // MB/s
	WriteSpeed float64 // MB/s
	Usage      float64 // Usage percentage
}

var (
	cachedCPUInfo    *CPUInfo
	cachedDiskInfo   []*DiskInfo
	cacheInitMutex   sync.Once
	diskInfoMutex    sync.RWMutex
	lastDiskUpdate   time.Time
	diskUpdatePeriod = 1 * time.Second

	// 无锁读取用原子存储
	diskInfoStore atomic.Value // []*DiskInfo

	// 渲染门控
	renderAccessMutex   sync.Mutex
	lastRenderAccess    time.Time
	renderAccessTimeout = 5 * time.Second

	diskSamplerOnce sync.Once
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
			}
		}()
	})
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
