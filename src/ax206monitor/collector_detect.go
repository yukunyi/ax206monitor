package main

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
)

func detectCPUInfo() *CPUInfo {
	info := &CPUInfo{
		Model:        "Unknown CPU",
		Cores:        runtime.NumCPU(),
		Threads:      runtime.NumCPU(),
		Architecture: runtime.GOARCH,
		Vendor:       "unknown",
	}

	if threads, err := cpu.Counts(true); err == nil && threads > 0 {
		info.Threads = threads
	}
	if cores, err := cpu.Counts(false); err == nil && cores > 0 {
		info.Cores = cores
	} else if info.Threads > 0 {
		info.Cores = info.Threads
	}

	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		var (
			maxFreq float64
			minFreq float64
		)
		for _, item := range cpuInfos {
			if strings.TrimSpace(item.ModelName) != "" && info.Model == "Unknown CPU" {
				info.Model = strings.TrimSpace(item.ModelName)
			}
			if strings.TrimSpace(item.VendorID) != "" && info.Vendor == "unknown" {
				info.Vendor = strings.TrimSpace(item.VendorID)
			}
			if item.Mhz > 0 {
				if maxFreq == 0 || item.Mhz > maxFreq {
					maxFreq = item.Mhz
				}
				if minFreq == 0 || item.Mhz < minFreq {
					minFreq = item.Mhz
				}
			}
		}
		info.MaxFreq = maxFreq
		info.MinFreq = minFreq
	}

	return info
}

func detectDiskInfo() []*DiskInfo {
	return detectDiskInfoByGopsutil()
}

func detectDiskInfoByGopsutil() []*DiskInfo {
	partitions, err := disk.Partitions(false)
	if err != nil {
		logWarnModule("disk", "disk partition detection failed: %v", err)
		return []*DiskInfo{}
	}

	items := make(map[string]*DiskInfo)
	for _, part := range partitions {
		name := normalizeDiskName(part.Device, part.Mountpoint)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := items[key]; exists {
			continue
		}

		usageTarget := strings.TrimSpace(part.Mountpoint)
		if usageTarget == "" {
			usageTarget = strings.TrimSpace(part.Device)
		}
		if usageTarget == "" {
			continue
		}

		usage, err := disk.Usage(usageTarget)
		if err != nil {
			continue
		}

		items[key] = &DiskInfo{
			Name:        name,
			Model:       inferDiskModel(part.Device, part.Mountpoint),
			Size:        int64(usage.Total / (1024 * 1024 * 1024)),
			Temperature: 0,
			ReadSpeed:   0,
			WriteSpeed:  0,
			Usage:       usage.UsedPercent,
		}
	}

	disks := make([]*DiskInfo, 0, len(items))
	for _, diskInfo := range items {
		disks = append(disks, diskInfo)
	}
	sort.Slice(disks, func(i, j int) bool {
		return disks[i].Name < disks[j].Name
	})
	return disks
}

func normalizeDiskName(device, mountpoint string) string {
	device = strings.TrimSpace(device)
	mountpoint = strings.TrimSpace(mountpoint)

	if device != "" {
		if runtime.GOOS == "windows" && strings.HasPrefix(strings.ToLower(device), `\\?\\volume`) && mountpoint != "" {
			return mountpoint
		}
		return device
	}
	if mountpoint != "" {
		return mountpoint
	}
	return ""
}

func inferDiskModel(device, mountpoint string) string {
	name := strings.TrimSpace(device)
	if name == "" {
		name = strings.TrimSpace(mountpoint)
	}
	if name == "" {
		return "Disk"
	}
	base := filepath.Base(name)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "Disk"
	}
	return strings.ToUpper(base)
}
