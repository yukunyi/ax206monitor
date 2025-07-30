package main

import (
	"sync"
	"time"
)

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
	initializeCache()
	return cachedDiskInfo
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
	// This would need platform-specific implementation
	// For now, return a placeholder value
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
	// This would need platform-specific implementation
	// For now, return a placeholder value
	return 0.0
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
	// This would need platform-specific implementation
	// For now, return a placeholder value
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
