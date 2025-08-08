package main

import (
	"sync"
	"time"
)

// NewDisk1TempMonitor creates a disk1 temperature monitor
func NewDisk1TempMonitor() MonitorItem {
	return CreateDiskMonitorByIndex(1, "temp", "°C", func(disk *DiskInfo) interface{} {
		return disk.Temperature
	})
}

// NewDisk1ReadSpeedMonitor creates a disk1 read speed monitor
func NewDisk1ReadSpeedMonitor() MonitorItem {
	return CreateDiskMonitorByIndex(1, "read_speed", "MB/s", func(disk *DiskInfo) interface{} {
		return disk.ReadSpeed
	})
}

// NewDisk1WriteSpeedMonitor creates a disk1 write speed monitor
func NewDisk1WriteSpeedMonitor() MonitorItem {
	return CreateDiskMonitorByIndex(1, "write_speed", "MB/s", func(disk *DiskInfo) interface{} {
		return disk.WriteSpeed
	})
}

// NewDisk1UsageMonitor creates a disk1 usage monitor
func NewDisk1UsageMonitor() MonitorItem {
	return CreateDiskMonitorByIndex(1, "usage", "%", func(disk *DiskInfo) interface{} {
		return disk.Usage
	})
}

// NewDisk1ModelMonitor creates a disk1 model monitor
func NewDisk1ModelMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateStringMonitor("disk1_model", "Disk 1 Model", func() (string, bool) {
		initializeCache()
		if len(cachedDiskInfo) > 0 {
			return cachedDiskInfo[0].Model, true
		}
		return "Unknown", false
	})
}

// NewDisk1NameMonitor creates a disk1 name monitor
func NewDisk1NameMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateStringMonitor("disk1_name", "Disk 1 Name", func() (string, bool) {
		initializeCache()
		if len(cachedDiskInfo) > 0 {
			return cachedDiskInfo[0].Name, true
		}
		return "Unknown", false
	})
}

// DiskIOStats represents disk I/O statistics
type DiskIOStats struct {
	ReadBytes    uint64
	WriteBytes   uint64
	ReadOps      uint64
	WriteOps     uint64
	ReadTime     uint64 // Time spent reading (ms)
	WriteTime    uint64 // Time spent writing (ms)
	IOTime       uint64 // Time spent doing I/Os (ms)
	WeightedTime uint64 // Weighted time spent doing I/Os (ms)
	Timestamp    time.Time
}

// DiskLatencyStats represents disk latency statistics
type DiskLatencyStats struct {
	ReadLatency  float64 // Average read latency in ms
	WriteLatency float64 // Average write latency in ms
	IOLatency    float64 // Average I/O latency in ms
}

var (
	diskIOStatsMutex sync.RWMutex
	lastDiskIOStats  map[string]*DiskIOStats
)

// getDiskReadSpeed calculates current disk read speed in MB/s
func getDiskReadSpeed() float64 {
	stats := getCurrentDiskIOStats()
	if len(stats) == 0 {
		return -1
	}

	var totalReadSpeed float64
	count := 0

	for _, stat := range stats {
		if stat.ReadSpeed > 0 {
			totalReadSpeed += stat.ReadSpeed
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return totalReadSpeed / float64(count)
}

// getDiskWriteSpeed calculates current disk write speed in MB/s
func getDiskWriteSpeed() float64 {
	stats := getCurrentDiskIOStats()
	if len(stats) == 0 {
		return -1
	}

	var totalWriteSpeed float64
	count := 0

	for _, stat := range stats {
		if stat.WriteSpeed > 0 {
			totalWriteSpeed += stat.WriteSpeed
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return totalWriteSpeed / float64(count)
}

// getCurrentDiskIOStats gets current disk I/O statistics
func getCurrentDiskIOStats() []*DiskInfo {
	return getCachedDiskInfo()
}

// updateDiskIOStats updates disk I/O statistics for speed calculation
func updateDiskIOStats() {
	diskIOStatsMutex.Lock()
	defer diskIOStatsMutex.Unlock()

	if lastDiskIOStats == nil {
		lastDiskIOStats = make(map[string]*DiskIOStats)
	}

	// This function would be called periodically to update disk I/O stats
	// The actual implementation would read from /proc/diskstats on Linux
	// or use platform-specific APIs on other systems
}

// DiskLatencyMonitor displays disk latency
type DiskLatencyMonitor struct {
	*BaseMonitorItem
}

func NewDiskLatencyMonitor() *DiskLatencyMonitor {
	return &DiskLatencyMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_latency",
			"Disk Latency",
			0, 0,
			"ms",
			2,
		),
	}
}

func (d *DiskLatencyMonitor) Update() error {
	latency := getDiskLatency()
	if latency >= 0 {
		d.SetValue(latency)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// getDiskLatency calculates average disk latency
func getDiskLatency() float64 {
	// For now, return a basic estimation based on disk activity
	// Real latency calculation would require parsing /proc/diskstats over time
	stats := getCurrentDiskIOStats()
	if len(stats) == 0 {
		return 0.0
	}

	// Simple estimation: if disks are active (read/write speed > 0), assume some latency
	var activeDisks int
	for _, stat := range stats {
		if stat.ReadSpeed > 0 || stat.WriteSpeed > 0 {
			activeDisks++
		}
	}

	if activeDisks > 0 {
		// Return a basic latency estimate (1-10ms range)
		return 2.5 // Average latency estimate in ms
	}

	return 0.0
}

// DiskIOPSMonitor displays disk IOPS (Input/Output Operations Per Second)
type DiskIOPSMonitor struct {
	*BaseMonitorItem
}

func NewDiskIOPSMonitor() *DiskIOPSMonitor {
	return &DiskIOPSMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_iops",
			"Disk IOPS",
			0, 0,
			"ops/s",
			0,
		),
	}
}

func (d *DiskIOPSMonitor) Update() error {
	iops := getDiskIOPS()
	if iops >= 0 {
		d.SetValue(iops)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// getDiskIOPS calculates current disk IOPS
func getDiskIOPS() float64 {
	// Estimate IOPS based on disk activity
	stats := getCurrentDiskIOStats()
	if len(stats) == 0 {
		return 0.0
	}

	// Simple estimation based on read/write speeds
	var totalIOPS float64

	for _, stat := range stats {
		// Rough estimation: 1 MB/s ≈ 250 IOPS (assuming 4KB blocks)
		readIOPS := stat.ReadSpeed * 250
		writeIOPS := stat.WriteSpeed * 250
		totalIOPS += readIOPS + writeIOPS
	}

	return totalIOPS
}

// DiskUtilizationMonitor displays disk utilization percentage
type DiskUtilizationMonitor struct {
	*BaseMonitorItem
}

func NewDiskUtilizationMonitor() *DiskUtilizationMonitor {
	return &DiskUtilizationMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_utilization",
			"Disk Util",
			0, 100,
			"%",
			1,
		),
	}
}

func (d *DiskUtilizationMonitor) Update() error {
	utilization := getDiskUtilization()
	if utilization >= 0 {
		d.SetValue(utilization)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// getDiskUtilization calculates disk utilization percentage
func getDiskUtilization() float64 {
	// Estimate utilization based on disk activity
	stats := getCurrentDiskIOStats()
	if len(stats) == 0 {
		return 0.0
	}

	// Calculate average utilization across all disks
	var totalUtil float64
	var count int

	for _, stat := range stats {
		// Simple estimation based on read/write activity
		activity := stat.ReadSpeed + stat.WriteSpeed

		// Convert MB/s to utilization percentage (rough estimation)
		// Assume 100 MB/s = 100% utilization for a typical disk
		util := activity
		if util > 100 {
			util = 100
		}

		totalUtil += util
		count++
	}

	if count > 0 {
		return totalUtil / float64(count)
	}
	return 0.0
}

// DiskQueueDepthMonitor displays disk queue depth
type DiskQueueDepthMonitor struct {
	*BaseMonitorItem
}

func NewDiskQueueDepthMonitor() *DiskQueueDepthMonitor {
	return &DiskQueueDepthMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_queue_depth",
			"Queue Depth",
			0, 0,
			"",
			0,
		),
	}
}

func (d *DiskQueueDepthMonitor) Update() error {
	queueDepth := getDiskQueueDepth()
	if queueDepth >= 0 {
		d.SetValue(queueDepth)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// getDiskQueueDepth calculates current disk queue depth
func getDiskQueueDepth() float64 {
	// This would need platform-specific implementation
	// For now, return a placeholder value
	return 0.0
}

// DiskMaxTempMonitor displays maximum disk temperature across all disks
type DiskMaxTempMonitor struct {
	*BaseMonitorItem
}

func NewDiskMaxTempMonitor() *DiskMaxTempMonitor {
	return &DiskMaxTempMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_max_temp",
			"Max Disk Temp",
			0, 80,
			"°C",
			0,
		),
	}
}

func (d *DiskMaxTempMonitor) Update() error {
	maxTemp := getDiskMaxTemperature()
	if maxTemp > 0 {
		d.SetValue(maxTemp)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// getDiskMaxTemperature calculates maximum disk temperature across all disks
func getDiskMaxTemperature() float64 {
	diskInfo := getCachedDiskInfo()
	if len(diskInfo) == 0 {
		return 0.0
	}

	var maxTemp float64
	hasValidTemp := false

	for _, disk := range diskInfo {
		if disk.Temperature > 0 {
			if !hasValidTemp || disk.Temperature > maxTemp {
				maxTemp = disk.Temperature
				hasValidTemp = true
			}
		}
	}

	if hasValidTemp {
		return maxTemp
	}
	return 0.0
}

// DiskTotalReadSpeedMonitor displays total read speed across all disks
type DiskTotalReadSpeedMonitor struct {
	*BaseMonitorItem
}

func NewDiskTotalReadSpeedMonitor() *DiskTotalReadSpeedMonitor {
	return &DiskTotalReadSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_total_read_speed",
			"Total Read",
			0, 0,
			"", // Dynamic unit will be set in SetValue
			2,
		),
	}
}

func (d *DiskTotalReadSpeedMonitor) Update() error {
	totalSpeed := getDiskTotalReadSpeed()
	if totalSpeed >= 0 {
		d.SetDiskSpeedValue(totalSpeed)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// SetDiskSpeedValue sets the disk speed value with dynamic unit formatting
func (d *DiskTotalReadSpeedMonitor) SetDiskSpeedValue(speedMBps float64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	value, unit := formatDiskSpeed(speedMBps)
	d.value.Value = value
	d.value.Unit = unit
}

// DiskTotalWriteSpeedMonitor displays total write speed across all disks
type DiskTotalWriteSpeedMonitor struct {
	*BaseMonitorItem
}

func NewDiskTotalWriteSpeedMonitor() *DiskTotalWriteSpeedMonitor {
	return &DiskTotalWriteSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_total_write_speed",
			"Total Write",
			0, 0,
			"", // Dynamic unit will be set in SetValue
			2,
		),
	}
}

func (d *DiskTotalWriteSpeedMonitor) Update() error {
	totalSpeed := getDiskTotalWriteSpeed()
	if totalSpeed >= 0 {
		d.SetDiskSpeedValue(totalSpeed)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// SetDiskSpeedValue sets the disk speed value with dynamic unit formatting
func (d *DiskTotalWriteSpeedMonitor) SetDiskSpeedValue(speedMBps float64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	value, unit := formatDiskSpeed(speedMBps)
	d.value.Value = value
	d.value.Unit = unit
}

// formatDiskSpeed formats disk speed with appropriate unit and spacing (same as network)
func formatDiskSpeed(speedMBps float64) (float64, string) {
	if speedMBps >= 1.0 {
		return speedMBps, " MiB/s"
	} else if speedMBps >= 0.001 {
		return speedMBps * 1024, " KiB/s"
	} else {
		return speedMBps * 1024 * 1024, " B/s"
	}
}

// getDiskTotalReadSpeed calculates total read speed across all disks
func getDiskTotalReadSpeed() float64 {
	diskInfo := getCachedDiskInfo()
	if len(diskInfo) == 0 {
		return -1
	}

	var totalReadSpeed float64

	for _, disk := range diskInfo {
		totalReadSpeed += disk.ReadSpeed
	}

	// Always return the total, even if 0
	return totalReadSpeed
}

// getDiskTotalWriteSpeed calculates total write speed across all disks
func getDiskTotalWriteSpeed() float64 {
	diskInfo := getCachedDiskInfo()
	if len(diskInfo) == 0 {
		return -1
	}

	var totalWriteSpeed float64

	for _, disk := range diskInfo {
		totalWriteSpeed += disk.WriteSpeed
	}

	// Always return the total, even if 0
	return totalWriteSpeed
}
